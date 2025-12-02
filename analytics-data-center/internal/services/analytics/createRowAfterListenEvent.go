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
	a.checkColumnInTables(ctx, evtData.Before, after, databaseEvt, schemaEvt, tableEvt, schemaIds)
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

	// В DWH таблицы и схемы могут отличаться от событий OLTP.
	// Используем имя view как таблицу в DWH, а схему — public по умолчанию.
	dwhSchemaName := "public"
	var dwhTableName string

	viewCache := make(map[int]models.View)
	var expectedColumns []models.Column
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
					expectedColumns = append(expectedColumns, table.Columns...)
				}
			}
		}
	}

	if dwhTableName == "" {
		dwhTableName = tableEvt
		log.Warn("не удалось определить имя таблицы DWH из схемы, используем имя из события", slog.String("table", dwhTableName))
	}

	columns, err := a.DWHProvider.GetColumnsTables(ctx, dwhSchemaName, dwhTableName)
	if err != nil {
		log.Error("ошибка получения колонок", slog.String("error", err.Error()))
		return err
	}

	log.Debug("используются таблица и схема DWH", slog.String("schema", dwhSchemaName), slog.String("table", dwhTableName))

	// Преобразуем список в map для быстрого поиска
	actualCols := make(map[string]struct{}, len(columns))
	for _, col := range columns {
		actualCols[col] = struct{}{}
	}

	renameCandidate, err := renameheuristics.DetectRenameCandidate(ctx, renameheuristics.DetectorConfig{
		ActualDWHColumns:      actualCols,
		BeforeEvent:           before,
		AfterEvent:            after,
		Database:              databaseEvt,
		Schema:                schemaEvt,
		Table:                 tableEvt,
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
				delete(actualCols, renameCandidate.OldName)
				actualCols[renameCandidate.NewName] = struct{}{}
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

						if renameCandidate != nil && column.Name == renameCandidate.OldName {
							column.Name = renameCandidate.NewName
							column.IsDeleted = false
							changed = true
							continue
						}

						_, exists := actualCols[column.Name]
						if column.IsDeleted == exists {
							// Был is_deleted=false, а колонки нет — ставим true. Или наоборот.
							column.IsDeleted = !exists
							changed = true
							if column.IsDeleted {
								changedData := make(map[string]interface{})
								message := fmt.Sprintf("Уважаемый пользователь из таблицы %s была удалена колонка %s", table.Name, column.Name)
								changedData["columnCnahged"] = column.Name
								changedData["messageEmail"] = message

								eventAfterChangedTable := &models.Event{
									EventName: "TableChanged",
									EventData: changedData,
								}
								a.SMTPClient.EventQueueSMTP <- *eventAfterChangedTable
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
