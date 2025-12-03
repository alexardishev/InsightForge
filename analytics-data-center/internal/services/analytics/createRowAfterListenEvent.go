package serviceanalytics

import (
	"analyticDataCenter/analytics-data-center/internal/domain/models"
	sqlgenerator "analyticDataCenter/analytics-data-center/internal/lib/SQLGenerator"
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
	var schems []models.View
	finalRow := make(map[string]interface{})
	conflictKeys := make(map[string]struct{})

	log := a.log.With(slog.String("op", op))
	after := evtData.After
	databaseEvt := evtData.Source.DB
	schemaEvt := evtData.Source.Schema
	tableEvt := evtData.Source.Table

	log.Info("Начинаю создание новой строки", slog.String("op", evtData.Op))

	schemaIds, err := a.SchemaProvider.GetSchems(ctx, databaseEvt, schemaEvt, tableEvt)
	if err != nil {
		log.Error("ошибка получения схемы", slog.Any("ошибка", err))
		return err
	}
	if err := a.checkColumnInTables(ctx, evtData.Before, after, databaseEvt, schemaEvt, tableEvt, schemaIds); err != nil {
		return err
	}
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
		schems = append(schems, schema)
	}

	for _, schema := range schems {
		for _, source := range schema.Sources {
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
						val, ok := evtData.After[column.Name]
						if !ok {
							continue
						}

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
						// Переделать на вызов функций
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
								log.Warn("Ожидалась строка JSON, но получено другое", slog.String("column", column.Name))
								continue
							}

							var jsonMap map[string]interface{}
							if err := json.Unmarshal([]byte(valStr), &jsonMap); err != nil {
								log.Warn("Ошибка парсинга JSON-строки", slog.String("column", column.Name), slog.String("error", err.Error()))
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

					var conflictColumns []string
					for k := range conflictKeys {
						conflictColumns = append(conflictColumns, k)
					}

					err := a.DWHProvider.InsertOrUpdateTransactional(
						ctx,
						schema.Name, // имя таблицы = имя view
						finalRow,
						conflictColumns)
					if err != nil {
						log.Error("ошибка вставки/обновления", slog.String("error", err.Error()))
						return err
					}
				}
			}
		}
	}

	log.Info("Успешная вставка в таблицу")
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

	// Алгоритм сверки колонок теперь идёт по цепочке «OLTP → схема → DWH».
	// Пример: в OLTP колонка переименована из name в title.
	//  1. Читаем фактические колонки из OLTP (увидим title).
	//  2. Сопоставляем с колонками из схемы: если в схеме ещё прописан name,
	//     то фиксируем rename-кандидат name→title и подставляем новое имя в expectedColumns.
	//  3. Сравниваем expectedColumns с DWH: если в DWH ещё есть name, а title нет,
	//     пытаемся переименовать столбец; после успешного rename обновляем карту actualCols.
	//  4. Если колонки из схемы нет в OLTP вовсе (колонку удалили), ставим is_deleted=true
	//     и отправляем уведомление, но при простом переименовании is_deleted не ставится.

	// В DWH таблицы и схемы могут отличаться от событий OLTP.
	// Используем имя view как таблицу в DWH, а схему — public по умолчанию.
	dwhSchemaName := "public"
	var dwhTableName string

	viewCache := make(map[int]models.View)
	var expectedColumns []models.Column
	columnTypes := make(map[string]string)
	schemaRenames := make(map[int]map[string]string)

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

	for _, schemaId := range schemaIds {
		view, err := a.SchemaProvider.GetView(ctx, int64(schemaId))
		if err != nil {
			log.Warn("не удалось получить view", slog.String("error", err.Error()))
			continue
		}
		viewCache[schemaId] = view

		if dwhTableName == "" {
			dwhTableName = view.Name
		}

		for _, source := range view.Sources {
			if source.Name != databaseEvt {
				continue
			}
			for _, schema := range source.Schemas {
				if schema.Name != schemaEvt {
					continue
				}
				for _, table := range schema.Tables {
					if table.Name != tableEvt {
						continue
					}

					renames, _ := buildSchemaToOLTPCandidates(table.Columns, oltpColumnsMap)
					if len(renames) > 0 {
						if _, ok := schemaRenames[schemaId]; !ok {
							schemaRenames[schemaId] = make(map[string]string)
						}
						for oldName, newName := range renames {
							schemaRenames[schemaId][oldName] = newName
						}
					}

					for _, column := range table.Columns {
						colCopy := column
						originalName := strings.ToLower(column.Name)
						if colCopy.Type != "" {
							columnTypes[originalName] = normalizeColumnType(colCopy.Type)
						}
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
	}

	if dwhTableName == "" {
		dwhTableName = tableEvt
		log.Warn("не удалось определить имя таблицы DWH из схемы, используем имя из события", slog.String("table", dwhTableName))
	}

	columns, err := a.DWHProvider.GetColumnsTables(ctx, dwhSchemaName, strings.ToLower(dwhTableName))
	if err != nil {
		log.Error("ошибка получения колонок", slog.String("error", err.Error()))
		return err
	}

	log.Debug("используются таблица и схема DWH", slog.String("schema", dwhSchemaName), slog.String("table", dwhTableName))

	// Преобразуем список в map для быстрого поиска
	actualCols := make(map[string]struct{}, len(columns))
	for _, col := range columns {
		actualCols[strings.ToLower(col)] = struct{}{}
	}

	log.Info("снимок колонок", slog.Any("oltp", sortedMapKeys(oltpColumnsMap)), slog.Any("schema", collectColumnNames(expectedColumns)), slog.Any("dwh", lowerCaseColumns(columns)))

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
		log.Warn("ошибка определения переименования", slog.String("error", err.Error()))
	}

	if renameCandidate != nil {
		log.Info("обнаружено переименование колонки", slog.String("from", renameCandidate.OldName), slog.String("to", renameCandidate.NewName), slog.String("strategy", renameCandidate.Strategy))
		renameQuery, err := sqlgenerator.GenerateRenameColumnQuery(a.DWHDbName, dwhSchemaName, dwhTableName, renameCandidate.OldName, renameCandidate.NewName)
		if err != nil {
			log.Error("не удалось сгенерировать запрос переименования", slog.String("error", err.Error()))
		} else {
			if err := a.DWHProvider.RenameColumn(ctx, renameQuery); err != nil {
				log.Error("ошибка применения переименования в DWH", slog.String("error", err.Error()))
				// даже если DWH не обновился, продолжаем помечать is_deleted, чтобы не пропустить проблему
			} else {
				delete(actualCols, strings.ToLower(renameCandidate.OldName))
				actualCols[strings.ToLower(renameCandidate.NewName)] = struct{}{}
			}
		}
	}

	// Перебираем все связанные схемы
	for _, schemaId := range schemaIds {
		view, ok := viewCache[schemaId]
		if !ok {
			fetchedView, err := a.SchemaProvider.GetView(ctx, int64(schemaId))
			if err != nil {
				log.Warn("не удалось получить view", slog.String("error", err.Error()))
				continue
			}
			viewCache[schemaId] = fetchedView
			view = fetchedView
		}
		changed := false
		tableRenames := schemaRenames[schemaId]

		for si := range view.Sources {
			source := &view.Sources[si]
			if source.Name != databaseEvt {
				continue
			}
			for sci := range source.Schemas {
				schema := &source.Schemas[sci]
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

						if renameCandidate != nil && strings.EqualFold(column.Name, renameCandidate.OldName) {
							column.Name = renameCandidate.NewName
							column.IsDeleted = false
							changed = true
							continue
						}

						if newName, ok := findRenameTarget(tableRenames, column.Name); ok {
							if !strings.EqualFold(column.Name, newName) {
								column.Name = newName
								changed = true
							}
							column.IsDeleted = false
						}

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

						_, exists := actualCols[normName]
						if column.IsDeleted == exists {
							// Был is_deleted=false, а колонки нет — ставим true. Или наоборот.
							column.IsDeleted = !exists
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
				log.Error("ошибка обновления view после is_deleted", slog.String("error", err.Error()))
			}
		} else {
			log.Info("view не изменился, обновление не требуется")
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
