package serviceanalytics

import (
	"analyticDataCenter/analytics-data-center/internal/domain/models"
	sqlgenerator "analyticDataCenter/analytics-data-center/internal/lib/SQLGenerator"
	"analyticDataCenter/analytics-data-center/internal/storage"
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"sync"
)

type IndexTransfer struct {
	IndexTransfer models.IndexTransferTable
	storage       storage.OLTPDB
}

type IndexTransfers struct {
	IndexTransfers []IndexTransfer
}

func (a *AnalyticsDataCenterService) createTempTables(ctx context.Context, quries models.Queries) error {
	const op = "analytics.createTempTables"
	var errorCreate error
	log := a.log.With(
		slog.String("op", op),
	)

	for _, query := range quries.Queries {
		err := a.DWHProvider.CreateTempTable(ctx, query.Query, query.TableName)
		if err != nil {
			errorCreate = err
			log.Error("не удалось создать временные таблицы", slog.String("error", err.Error()))
			break

		}

	}
	if errorCreate != nil {
		for _, tableQuery := range quries.Queries {
			err := a.DWHProvider.DeleteTempTable(ctx, tableQuery.TableName)
			if err != nil {
				//TO DO сделать worker , который раз в какое-то время запускается и чистит темп таблицы на такие случаи.
				log.Error("не удалось удалить временную таблицу",
					slog.String("table", tableQuery.TableName),
					slog.String("error", err.Error()),
				)
			}
		}
		return errorCreate

	}

	return nil

}

func (a *AnalyticsDataCenterService) getCountInsertData(ctx context.Context, viewSchema models.View, tempTables []string) ([]models.CountInsertData, error) {
	const op = "analytics.getCountInsertData"
	var sliceCountInsertData []models.CountInsertData
	log := a.log.With(
		slog.String("op", op),
	)

	queries, err := sqlgenerator.GenerateCountQueries(viewSchema, log)
	if err != nil {
		log.Info("ошибка генерации запроса", slog.String("error", err.Error()))
		return []models.CountInsertData{}, err
	}

	for idx, query := range queries.Queries {
		oltpStorage, err := a.OLTPFactory.GetOLTPStorage(ctx, query.SourceName)
		if err != nil {
			log.Error("Невозможно подключиться к OLTP хранилищу", slog.String("error", err.Error()))
			return []models.CountInsertData{}, err
		}
		count, err := oltpStorage.GetCountInsertData(ctx, query.Query)
		if err != nil {
			log.Error("неудачная попытка посчитать количество данных", slog.String("error", err.Error()))
			return []models.CountInsertData{}, err
		}

		if count <= 0 {
			log.Info("нет данных для переноса", slog.String("table", query.TableName))
			continue
		}

		item := models.CountInsertData{
			TableName:     query.TableName,
			Count:         count,
			DataBaseName:  query.SourceName,
			TempTableName: tempTables[idx],
		}
		log.Info("готово", slog.String("TableName", item.TableName))
		log.Info("готово", slog.String("TempTableName", item.TempTableName))

		sliceCountInsertData = append(sliceCountInsertData, item)

	}
	log.Info("готово", slog.Int("tablesWithData", len(sliceCountInsertData)))

	return sliceCountInsertData, nil
}

func (a *AnalyticsDataCenterService) prepareAndInsertData(ctx context.Context, countData *[]models.CountInsertData, viewSchema *models.View) (bool, error) {
	const op = "analytics.prepareDataForInsert"
	log := a.log.With(
		slog.String("op", op),
	)
	var tempTbl []string
	var wg sync.WaitGroup
	var hasError bool
	var mu sync.Mutex
	runtime.GOMAXPROCS(2)

	for _, tempTableInsert := range *countData {
		log.Info("запуск вставки и подготовки данных", slog.String("Таблица", tempTableInsert.TableName))
		tempTbl = append(tempTbl, tempTableInsert.TempTableName)
		oltpStorage, err := a.OLTPFactory.GetOLTPStorage(ctx, tempTableInsert.DataBaseName)
		if err != nil {
			log.Error("Невозможно подключиться к OLTP хранилищу", slog.String("error", err.Error()))
			return false, err
		}
		procCount := a.calculateWorkerCount(tempTableInsert.Count)
		if runtime.NumCPU() <= 1 {
			procCount = 1
		}
		if tempTableInsert.Count <= 0 {
			continue
		}
		chunkSize := (tempTableInsert.Count + procCount - 1) / procCount // Округление вверх
		for i := int64(0); i < procCount; i++ {
			start := i * chunkSize
			end := (i + 1) * chunkSize
			if end > tempTableInsert.Count {
				end = tempTableInsert.Count
			}
			tableName := tempTableInsert.TableName
			sourceName := tempTableInsert.DataBaseName
			tempTableName := tempTableInsert.TempTableName

			wg.Add(1)
			go func(start, end int64, tableName, sourceName string, tempTableName string) {
				log.Info("Горутина запущена", slog.String("Для таблицы", tableName))
				defer wg.Done()
				query, err := sqlgenerator.GenerateSelectInsertDataQuery(*viewSchema, start, end, tableName, log)
				if err != nil {
					log.Error("ошибка генерации SQL", slog.String("error", err.Error()))
					mu.Lock()
					hasError = true
					mu.Unlock()
					return
				}
				insertData, err := oltpStorage.SelectDataToInsert(ctx, query.Query)
				if err != nil {
					log.Error("ошибка при SELECT из OLTP", slog.String("error", err.Error()))
					mu.Lock()
					hasError = true
					mu.Unlock()
					return
				}
				log.Info("получены данные", slog.Any("rows", len(insertData)))
				log.Info("Запрос для таблицы", slog.String("таблица", tempTableName))
				queryInsert, err := sqlgenerator.GeneratetInsertDataQuery(*viewSchema, insertData, tempTableName, log)
				if err != nil {
					log.Error("ошибка при генерации INSERT запроса", slog.String("error", err.Error()))
					mu.Lock()
					hasError = true
					mu.Unlock()
					return
				}

				log.Info("Запрос готов")
				err = a.DWHProvider.InsertDataToDWH(ctx, queryInsert.Query)
				if err != nil {
					log.Error("ошибка при вставки данных в таблицу", slog.String("error", err.Error()))
					mu.Lock()
					hasError = true
					mu.Unlock()
					return
				}
				log.Info("Вставка данных завершена")

			}(start, end, tableName, sourceName, tempTableName)

		}

	}
	wg.Wait()

	if hasError {
		a.DeleteTempTables(ctx, tempTbl)
		return false, fmt.Errorf("одна или несколько горутин завершились с ошибкой")
	}
	// TO DO сделать возможность указывать схему гибко через YAML
	viewJoin, err := a.prepareViewJoin(ctx, tempTbl, "public")
	if err != nil {
		a.DeleteTempTables(ctx, tempTbl)
		log.Error("Ошибка", slog.String("error", err.Error()))
		return false, err
	}
	query, err := sqlgenerator.CreateViewQuery(*viewSchema, *viewJoin, log)
	if err != nil {
		a.DeleteTempTables(ctx, tempTbl)
		log.Error("Ошибка", slog.String("error", err.Error()))
		return false, err
	}

	a.DWHProvider.MergeTempTables(ctx, query.Query)
	a.DeleteTempTables(ctx, tempTbl)
	return true, nil
}

func (a *AnalyticsDataCenterService) calculateWorkerCount(totalCount int64) int64 {
	switch {
	case totalCount > 500_000:
		return 8
	case totalCount > 200_000:
		return 6
	case totalCount > 100_000:
		return 4
	case totalCount > 10_000:
		return 2
	default:
		return 1
	}
}

func (a *AnalyticsDataCenterService) prepareViewJoin(ctx context.Context, tempTbl []string, schemaName string) (viewJoins *models.ViewJoinTable, err error) {
	const op = "analytics.prepareViewJoin"
	log := a.log.With(
		slog.String("op", op),
	)

	var tablesTemp []models.TempTable

	for _, tempTable := range tempTbl {
		columnsTempTables, err := a.DWHProvider.GetColumnsTables(ctx, schemaName, tempTable)
		if err != nil {
			log.Error("Невозможно получить колонки для временных таблиц", slog.String("error", err.Error()))
			return nil, err
		}
		var columnsTemp []models.TempColumn

		for _, col := range columnsTempTables {
			columnsTemp = append(columnsTemp, models.TempColumn{
				ColumnName: col,
			})
		}

		tempTables := &models.TempTable{
			TempTableName: tempTable,
			TempColumns:   columnsTemp,
		}
		tablesTemp = append(tablesTemp, *tempTables)
	}

	viewJoin := &models.ViewJoinTable{
		TempTables: tablesTemp,
	}
	return viewJoin, nil

}

func (a *AnalyticsDataCenterService) DeleteTempTables(ctx context.Context, tempTbl []string) error {
	const op = "analytics.DeleteTempTables"
	log := a.log.With(
		slog.String("op", op),
	)
	for _, temp := range tempTbl {
		err := a.DWHProvider.DeleteTempTable(ctx, temp)
		if err != nil {
			log.Error("Невозможно удалить таблицу", slog.String("error", err.Error()))
			return err
		}
	}
	return nil

}

func (a *AnalyticsDataCenterService) transferIndixesAndConstraint(ctx context.Context, viewSchema *models.View) error {
	const op = "analytics.transferIndicesAndConstraint"
	log := a.log.With(
		slog.String("op", op),
	)
	var OLTPSourceName []string
	var indexTransfers IndexTransfers
	for _, src := range viewSchema.Sources {
		storage, err := a.OLTPFactory.GetOLTPStorage(ctx, src.Name)
		if err != nil {
			log.Error("Невозможно получить хранилище OLTP", slog.String("error", err.Error()))
			return err
		}
		OLTPSourceName = append(OLTPSourceName, src.Name)
		for _, sch := range src.Schemas {
			for _, tbl := range sch.Tables {
				indexTransfer := &IndexTransfer{
					IndexTransfer: models.IndexTransferTable{
						TableName:  tbl.Name,
						SourceName: src.Name,
						SchemaName: sch.Name,
					},
					storage: storage,
				}
				indexTransfers.IndexTransfers = append(indexTransfers.IndexTransfers, *indexTransfer)
			}

		}
	}

	for _, transferTable := range indexTransfers.IndexTransfers {
		storage := transferTable.storage
		indexes, err := storage.GetIndexes(ctx, transferTable.IndexTransfer.TableName, transferTable.IndexTransfer.SchemaName)
		if err != nil {
			log.Error("Невозможно получить индексы таблиц", slog.String("error", err.Error()))
			return err
		}
		// constraints, err := storage.GetConstraint(ctx, transferTable.IndexTransfer.TableName, transferTable.IndexTransfer.SchemaName)
		// if err != nil {
		// 	log.Error("Невозможно получить ограничения таблиц", slog.String("error", err.Error()))
		// 	return err
		// }

		// for _, constraint := range constraints.Constraints {
		// 	query := sqlgenerator.TransformConstraintToExpression(constraint, "public", viewSchema.Name, a.log)
		// 	err = a.DWHProvider.CreateConstraint(ctx, query)
		// 	if err != nil {
		// 		return err
		// 	}
		// }
		for _, index := range indexes.Indexes {
			// TO DO инжектировать в сервис DWH схему, если она нужна через config
			query, err := sqlgenerator.TransformIndexDefToSQLExpression(index, transferTable.IndexTransfer.SchemaName, transferTable.IndexTransfer.TableName, "public", viewSchema.Name, a.log)
			// log.Info("Вот запрос на создание индекса в новой таблице", slog.String("query", query))
			if err != nil {
				log.Error("Невозможно сформировать запрос на создание индексов", slog.String("error", err.Error()))
				return err
			}
			err = a.DWHProvider.CreateIndex(ctx, query)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
