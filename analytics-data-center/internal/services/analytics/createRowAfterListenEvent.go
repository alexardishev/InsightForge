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

	_ = before
	_ = after

	dwhSchemaName := "public"

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

					for _, column := range table.Columns {
						colCopy := column
						colCopy.Name = strings.ToLower(colCopy.Name)

						if colCopy.Type != "" {
							columnTypes[colCopy.Name] = normalizeColumnType(colCopy.Type)
						}

						if colCopy.Type == "" {
							if oltpCol, exists := oltpColumnsMap[colCopy.Name]; exists {
								colCopy.Type = oltpCol.Type
							}
						}

						if _, ok := columnTypes[colCopy.Name]; !ok {
							columnTypes[colCopy.Name] = normalizeColumnType(colCopy.Type)
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

		schemaCols := make(map[string]models.Column)
		for _, col := range expectedColumns {
			schemaCols[col.Name] = col
		}

		schemaOnly := make([]string, 0)
		missingInDWH := make([]string, 0)
		dwhOnly := make([]string, 0)
		oldCandidates := make([]string, 0)
		newCandidates := make([]string, 0)

		for name := range schemaCols {
			if _, ok := oltpColumnsMap[name]; !ok {
				schemaOnly = append(schemaOnly, name)
			}
		}

		for name := range oltpColumnsMap {
			if _, ok := schemaCols[name]; ok {
				if _, exists := actualCols[name]; !exists {
					missingInDWH = append(missingInDWH, name)
				}
			}
		}

		for name := range actualCols {
			if _, ok := schemaCols[name]; !ok {
				dwhOnly = append(dwhOnly, name)
			}
		}

		for name := range schemaCols {
			if _, inDWH := actualCols[name]; inDWH {
				if _, inOLTP := oltpColumnsMap[name]; !inOLTP {
					oldCandidates = append(oldCandidates, name)
				}
			}
		}

		for _, name := range missingInDWH {
			newCandidates = append(newCandidates, name)
		}

		renameCandidates, err := renameheuristics.DetectRenameCandidate(ctx, renameheuristics.DetectorConfig{
			OldCandidates: oldCandidates,
			NewCandidates: newCandidates,
			ColumnTypes:   columnTypes,
			Logger:        a.log.Logger,
		})
		if err != nil {
			log.Warn("ошибка определения переименования",
				slog.String("error", err.Error()),
				slog.String("view", viewName))
		}

		var items []models.ColumnMismatchItem

		for _, name := range schemaOnly {
			n := name
			items = append(items, models.ColumnMismatchItem{OldColumnName: &n, Type: models.ColumnMismatchTypeSchemaOnly})
		}

		for _, name := range missingInDWH {
			n := name
			items = append(items, models.ColumnMismatchItem{NewColumnName: &n, Type: models.ColumnMismatchTypeMissingInDWH})
		}

		for _, name := range dwhOnly {
			n := name
			items = append(items, models.ColumnMismatchItem{OldColumnName: &n, Type: models.ColumnMismatchTypeDWHOnly})
		}

		for _, candidate := range renameCandidates {
			oldName := candidate.OldName
			newName := candidate.NewName
			score := candidate.Score
			items = append(items, models.ColumnMismatchItem{
				OldColumnName: &oldName,
				NewColumnName: &newName,
				Score:         &score,
				Type:          models.ColumnMismatchTypeRename,
			})
		}

		if len(items) == 0 {
			log.Info("несоответствий не найдено", slog.String("view", viewName))
			continue
		}

		openGroup, err := a.ColumnMismatchStorage.GetOpenMismatchGroup(ctx, int64(schemaId), databaseEvt, schemaEvt, tableEvt)
		if err != nil {
			if errors.Is(err, storage.ErrMismatchNotFound) {
				group := models.ColumnMismatchGroup{
					SchemaID:     int64(schemaId),
					DatabaseName: databaseEvt,
					SchemaName:   schemaEvt,
					TableName:    tableEvt,
					Status:       models.ColumnMismatchStatusOpen,
				}
				if _, err := a.ColumnMismatchStorage.CreateMismatchGroup(ctx, group, items); err != nil {
					log.Error("не удалось создать группу рассинхронов", slog.String("error", err.Error()), slog.String("view", viewName))
					return err
				}
				log.Warn("создана новая группа рассинхронов", slog.String("view", viewName))
				continue
			}
			return err
		}

		if err := a.ColumnMismatchStorage.ReplaceMismatchItems(ctx, openGroup.Group.ID, items); err != nil {
			log.Error("не удалось обновить элементы рассинхрона", slog.String("error", err.Error()), slog.String("view", viewName))
			return err
		}
		log.Warn("обновлены элементы существующей группы рассинхронов", slog.String("view", viewName))
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
