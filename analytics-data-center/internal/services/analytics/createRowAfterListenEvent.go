package serviceanalytics

import (
	"analyticDataCenter/analytics-data-center/internal/domain/models"
	renameheuristics "analyticDataCenter/analytics-data-center/internal/lib/renameheuristics"
	"analyticDataCenter/analytics-data-center/internal/storage"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"github.com/adrg/strutil/metrics"
)

func (a *AnalyticsDataCenterService) createRowAfterListenEventInDWH(ctx context.Context, evtData models.CDCEventData) error {
	const op = "createRowAfterListenEventInDWH"
	type viewInfo struct {
		id   int
		view models.View
	}
	var schems []viewInfo

	log := a.log.With(slog.String("op", op))

	after := evtData.After
	databaseEvt := evtData.Source.DB
	schemaEvt := evtData.Source.Schema
	tableEvt := evtData.Source.Table

	log.Info("Начинаю создание новой строки", slog.String("op", evtData.Op))

	// 1. Находим все схемы (view), привязанные к этой таблице
	schemaIds, err := a.SchemaProvider.GetSchems(ctx, databaseEvt, schemaEvt, tableEvt)
	if err != nil {
		log.Error("ошибка получения схемы", slog.Any("ошибка", err))
		return err
	}

	// 2. Проверяем/переименовываем колонки (по каждой вьюхе внутри)
	if err := a.checkColumnInTables(ctx, evtData.Before, after, databaseEvt, schemaEvt, tableEvt, schemaIds); err != nil {
		return err
	}

	// 3. Забираем сами view
	for _, schemaId := range schemaIds {
		schema, err := a.SchemaProvider.GetView(ctx, int64(schemaId))
		if err != nil {
			if errors.Is(err, storage.ErrSchemaNotFound) {
				log.Warn("view not found", slog.String("error", err.Error()))
				return fmt.Errorf("%s: %s", op, ErrInvalidSchemID)
			}
			log.Warn("ошибка получения схемы", slog.String("error", err.Error()))
			return fmt.Errorf("%s: %s", op, err)
		}
		schems = append(schems, viewInfo{id: schemaId, view: schema})
	}

	// 4. Для КАЖДОЙ view отдельно собираем finalRow и conflictKeys
	for _, schema := range schems {
		viewName := schema.view.Name

		hasSuggestion, err := a.RenameSuggestionStorage.HasSuggestion(ctx, int64(schema.id), databaseEvt, schemaEvt, tableEvt)
		if err != nil {
			log.Error("ошибка проверки предложений переименования", slog.String("error", err.Error()), slog.String("view", viewName))
			return err
		}

		if hasSuggestion {
			log.Warn("Пропускаю вставку — есть предложение по переименованию", slog.String("view", viewName), slog.String("table", tableEvt))
			continue
		}

		finalRow := make(map[string]interface{})
		conflictKeys := make(map[string]struct{})
		hasData := false

		for _, source := range schema.view.Sources {
			if source.Name != databaseEvt {
				continue
			}
			for _, sch := range source.Schemas {
				if sch.Name != schemaEvt {
					continue
				}
				for _, table := range sch.Tables {
					if table.Name != tableEvt {
						continue
					}

					for _, column := range table.Columns {
						val, ok := after[column.Name]
						if !ok {
							continue
						}
						hasData = true

						// всегда кладём по имени колонки
						finalRow[column.Name] = val

						// если ключевая — добавляем её в conflictKeys
						if column.IsUpdateKey {
							conflictKeys[column.Name] = struct{}{}
							if column.ViewKey != "" && column.ViewKey != column.Name {
								finalRow[column.ViewKey] = val
								conflictKeys[column.ViewKey] = struct{}{}
							}
						}

						// === FieldTransform ===
						if column.Transform != nil && column.Transform.Type == "FieldTransform" && column.Transform.Mapping != nil {
							rawStr := fmt.Sprintf("%v", val)
							if transformed, ok := column.Transform.Mapping.Mapping[rawStr]; ok {
								outputColumn := column.Transform.OutputColumn
								if outputColumn == "" {
									outputColumn = column.Name + "_transformed"
								}
								finalRow[outputColumn] = transformed
							}
						}

						// === JSON Transform ===
						if column.Transform != nil && column.Transform.Type == "JSON" && column.Transform.Mapping != nil {
							valStr, ok := val.(string)
							if !ok {
								log.Warn("Ожидалась строка JSON, но получено другое",
									slog.String("column", column.Name),
									slog.String("view", viewName))
								continue
							}

							var jsonMap map[string]interface{}
							if err := json.Unmarshal([]byte(valStr), &jsonMap); err != nil {
								log.Warn("Ошибка парсинга JSON-строки",
									slog.String("column", column.Name),
									slog.String("view", viewName),
									slog.String("error", err.Error()))
								continue
							}

							for _, mappingJSON := range column.Transform.Mapping.MappingJSON {
								for jsonField, outputColumn := range mappingJSON.Mapping {
									if extractedVal, exists := jsonMap[jsonField]; exists {
										finalRow[outputColumn] = extractedVal
									}
								}
							}
						}
					}
				}
			}
		}

		if !hasData {
			log.Info("для view нет колонок по этому событию, вставка пропущена",
				slog.String("view", viewName))
			continue
		}

		var conflictColumns []string
		for k := range conflictKeys {
			conflictColumns = append(conflictColumns, k)
		}

		if err := a.DWHProvider.InsertOrUpdateTransactional(
			ctx,
			viewName, // таблица в DWH = имя view
			finalRow,
			conflictColumns,
		); err != nil {
			log.Error("ошибка вставки/обновления",
				slog.String("error", err.Error()),
				slog.String("view", viewName))
			return err
		}
	}

	log.Info("Успешная вставка в таблицы DWH по всем view")
	return nil
}

func (a *AnalyticsDataCenterService) checkColumnInTables(
	ctx context.Context,
	before map[string]interface{},
	after map[string]interface{},
	databaseEvt string,
	schemaEvt string,
	tableEvt string,
	schemaIds []int,
) error {
	const op = "checkColumnInTables"
	log := a.log.With(slog.String("op", op))

	dwhSchemaName := "public"

	// 1. Колонки из OLTP — "истина" для этой таблицы
	oltpStorage, err := a.OLTPFactory.GetOLTPStorage(ctx, databaseEvt)
	if err != nil {
		log.Error("ошибка получения OLTP", slog.String("error", err.Error()))
		return err
	}
	oltpColumns, err := oltpStorage.GetColumns(ctx, schemaEvt, tableEvt)
	if err != nil {
		log.Error("ошибка получения колонок OLTP", slog.String("error", err.Error()))
		return err
	}
	oltpColumnsMap := normalizeColumnsToMap(oltpColumns)

	// 2. Обрабатываем КАЖДЫЙ view (schemaId) отдельно
	for _, schemaId := range schemaIds {
		view, err := a.SchemaProvider.GetView(ctx, int64(schemaId))
		if err != nil {
			log.Warn("не удалось получить view", slog.String("error", err.Error()), slog.Int("schemaId", schemaId))
			continue
		}

		viewName := view.Name
		dwhTableName := viewName
		if dwhTableName == "" {
			dwhTableName = tableEvt
			log.Warn("не удалось определить имя таблицы DWH из схемы, используем имя из события",
				slog.String("table", dwhTableName),
				slog.String("view", viewName))
		}

		// 2.1. Колонки таблицы DWH для ЭТОГО view
		columns, err := a.DWHProvider.GetColumnsTables(ctx, dwhSchemaName, strings.ToLower(dwhTableName))
		if err != nil {
			log.Error("ошибка получения колонок DWH",
				slog.String("error", err.Error()),
				slog.String("view", viewName))
			return err
		}

		actualCols := make(map[string]struct{}, len(columns))
		for _, col := range columns {
			actualCols[strings.ToLower(col)] = struct{}{}
		}

		log.Debug("используются таблица и схема DWH для view",
			slog.String("schema", dwhSchemaName),
			slog.String("table", dwhTableName),
			slog.String("view", viewName))

		var expectedColumns []models.Column
		columnTypes := make(map[string]string)

		// 3. Собираем колонки текущего view для нужной таблицы
		for _, source := range view.Sources {
			if source.Name != databaseEvt {
				continue
			}
			for _, sch := range source.Schemas {
				if sch.Name != schemaEvt {
					continue
				}
				for _, table := range sch.Tables {
					if table.Name != tableEvt {
						continue
					}

					// кандидаты rename между схемой и OLTP — используем ТОЛЬКО
					// для расчёта expectedColumns, НЕ для изменения view
					renames, _ := buildSchemaToOLTPCandidates(table.Columns, oltpColumnsMap)

					for _, column := range table.Columns {
						colCopy := column
						originalName := strings.ToLower(column.Name)

						if colCopy.Type != "" {
							columnTypes[originalName] = normalizeColumnType(colCopy.Type)
						}

						// Для эвристики: если нашли переименование на уровне schema↔OLTP,
						// в expectedColumns кладём новое имя, НО НЕ меняем сам view.
						if newName, ok := renames[column.Name]; ok {
							colCopy.Name = strings.ToLower(newName)
							if colCopy.Type == "" {
								if oltpCol, exists := oltpColumnsMap[strings.ToLower(newName)]; exists {
									colCopy.Type = oltpCol.Type
								}
							}
						} else {
							colCopy.Name = strings.ToLower(colCopy.Name)
							if colCopy.Type == "" {
								if oltpCol, exists := oltpColumnsMap[colCopy.Name]; exists {
									colCopy.Type = oltpCol.Type
								}
							}
						}

						if _, ok := columnTypes[strings.ToLower(colCopy.Name)]; !ok {
							columnTypes[strings.ToLower(colCopy.Name)] = normalizeColumnType(colCopy.Type)
						}

						expectedColumns = append(expectedColumns, colCopy)
					}
				}
			}
		}

		log.Info("снимок колонок для view",
			slog.String("view", viewName),
			slog.Any("oltp", sortedMapKeys(oltpColumnsMap)),
			slog.Any("schema", collectColumnNames(expectedColumns)),
			slog.Any("dwh", lowerCaseColumns(columns)))

		// 4. Пытаемся найти rename-кандидат ДЛЯ ЭТОГО view
		renameCandidate, err := renameheuristics.DetectRenameCandidate(ctx, renameheuristics.DetectorConfig{
			ActualDWHColumns:      actualCols,
			BeforeEvent:           before,
			AfterEvent:            after,
			Database:              databaseEvt,
			Schema:                schemaEvt,
			Table:                 tableEvt,
			ColumnTypes:           columnTypes,
			RenameHeuristicEnable: a.RenameHeuristicEnabled,
			ExpectedColumns:       expectedColumns,
			Logger:                a.log.Logger,
		})
		if err != nil {
			log.Warn("ошибка определения переименования",
				slog.String("error", err.Error()),
				slog.String("view", viewName))
		}

		if renameCandidate != nil {
			log.Info("обнаружено переименование колонки",
				slog.String("from", renameCandidate.OldName),
				slog.String("to", renameCandidate.NewName),
				slog.String("strategy", renameCandidate.Strategy),
				slog.String("view", viewName))

			suggestion := models.ColumnRenameSuggestion{
				SchemaID:      int64(schemaId),
				DatabaseName:  databaseEvt,
				SchemaName:    schemaEvt,
				TableName:     tableEvt,
				OldColumnName: renameCandidate.OldName,
				NewColumnName: renameCandidate.NewName,
				Strategy:      renameCandidate.Strategy,
			}

			if err := a.RenameSuggestionStorage.CreateSuggestion(ctx, suggestion); err != nil {
				log.Error("не удалось сохранить предложение переименования",
					slog.String("error", err.Error()),
					slog.String("view", viewName))
			}

			log.Warn("view пропущен из-за предложения переименования", slog.String("view", viewName))
			// Никаких изменений схемы, просто пропускаем дальнейшую обработку этого view
			continue
		}

		// 5. Обновляем сам view: ТОЛЬКО is_deleted, без каких-либо rename по имени
		changed := false

		for si := range view.Sources {
			source := &view.Sources[si]
			if source.Name != databaseEvt {
				continue
			}
			for sci := range source.Schemas {
				schema := &view.Sources[si].Schemas[sci]
				if schema.Name != schemaEvt {
					continue
				}
				for ti := range schema.Tables {
					table := &schema.Tables[ti]
					if table.Name != tableEvt {
						continue
					}

					for ci := range table.Columns {
						column := &table.Columns[ci]

						normName := strings.ToLower(column.Name)
						_, existsInOLTP := oltpColumnsMap[normName]
						if !existsInOLTP {
							if !column.IsDeleted {
								column.IsDeleted = true
								changed = true
								a.sendColumnRemovedEvent(table.Name, column.Name)
							}
							continue
						}

						_, existsInDWH := actualCols[normName]
						if column.IsDeleted == existsInDWH {
							// Был is_deleted=false, а колонки нет — ставим true. Или наоборот.
							column.IsDeleted = !existsInDWH
							changed = true
							if column.IsDeleted {
								a.sendColumnRemovedEvent(table.Name, column.Name)
							}
						}
					}
				}
			}
		}

		if changed {
			if err := a.SchemaProvider.UpdateView(ctx, view, schemaId); err != nil {
				log.Error("ошибка обновления view после is_deleted",
					slog.String("error", err.Error()),
					slog.String("view", viewName))
			}
		} else {
			log.Info("view не изменился, обновление не требуется",
				slog.String("view", viewName))
		}
	}

	return nil
}

func normalizeColumnsToMap(columns []models.Column) map[string]models.Column {
	result := make(map[string]models.Column, len(columns))

	for _, col := range columns {
		result[strings.ToLower(col.Name)] = col
	}

	return result
}

func buildSchemaToOLTPCandidates(schemaColumns []models.Column, oltpColumns map[string]models.Column) (map[string]string, map[string]struct{}) {
	renames := make(map[string]string)
	deleted := make(map[string]struct{})

	schemaNames := make(map[string]struct{}, len(schemaColumns))
	for _, col := range schemaColumns {
		schemaNames[strings.ToLower(col.Name)] = struct{}{}
	}

	var oltpOnly []models.Column
	for name, col := range oltpColumns {
		if _, ok := schemaNames[name]; !ok {
			oltpOnly = append(oltpOnly, col)
		}
	}

	for _, col := range schemaColumns {
		normName := strings.ToLower(col.Name)
		if _, ok := oltpColumns[normName]; ok {
			continue
		}

		if newName, ok := findBestRenameCandidate(col, oltpOnly); ok {
			renames[col.Name] = newName
			for i, c := range oltpOnly {
				if strings.EqualFold(c.Name, newName) {
					oltpOnly = append(oltpOnly[:i], oltpOnly[i+1:]...)
					break
				}
			}
			continue
		}

		deleted[col.Name] = struct{}{}
	}

	return renames, deleted
}

func findBestRenameCandidate(schemaColumn models.Column, candidates []models.Column) (string, bool) {
	jw := metrics.NewJaroWinkler()
	oldType := normalizeColumnType(schemaColumn.Type)
	var bestName string
	bestScore := 0.0

	minSimilarity := 0.45

	for _, candidate := range candidates {
		newType := normalizeColumnType(candidate.Type)
		if oldType != "" && newType != "" && oldType != newType {
			continue
		}

		score := jw.Compare(normalizeColumnName(schemaColumn.Name), normalizeColumnName(candidate.Name))
		if oldType != "" && newType == oldType {
			score += 0.1
		}

		if score < minSimilarity {
			continue
		}

		if bestName == "" || score > bestScore {
			bestName = candidate.Name
			bestScore = score
		}
	}

	return bestName, bestName != ""
}

func normalizeColumnName(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "_", " ")
	return strings.TrimSpace(s)
}

func normalizeColumnType(t string) string {
	return strings.ToLower(strings.TrimSpace(t))
}

func sortedMapKeys(columns map[string]models.Column) []string {
	keys := make([]string, 0, len(columns))
	for name := range columns {
		keys = append(keys, name)
	}
	sort.Strings(keys)
	return keys
}

func collectColumnNames(columns []models.Column) []string {
	names := make([]string, 0, len(columns))
	for _, col := range columns {
		names = append(names, col.Name)
	}
	sort.Strings(names)
	return names
}

func lowerCaseColumns(columns []string) []string {
	result := make([]string, 0, len(columns))
	for _, col := range columns {
		result = append(result, strings.ToLower(col))
	}
	sort.Strings(result)
	return result
}

func findRenameTarget(renames map[string]string, name string) (string, bool) {
	for oldName, newName := range renames {
		if strings.EqualFold(oldName, name) {
			return newName, true
		}
	}
	return "", false
}

func (a *AnalyticsDataCenterService) sendColumnRemovedEvent(tableName, columnName string) {
	if a.SMTPClient.EventQueueSMTP == nil {
		return
	}

	changedData := make(map[string]interface{})
	message := fmt.Sprintf("Уважаемый пользователь из таблицы %s была удалена колонка %s", tableName, columnName)
	changedData["columnCnahged"] = columnName
	changedData["messageEmail"] = message

	eventAfterChangedTable := models.Event{
		EventName: "TableChanged",
		EventData: changedData,
	}

	select {
	case a.SMTPClient.EventQueueSMTP <- eventAfterChangedTable:
	default:
		a.log.Warn("очередь уведомлений переполнена, событие TableChanged пропущено")
	}
}
