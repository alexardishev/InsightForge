package serviceanalytics

import (
	"analyticDataCenter/analytics-data-center/internal/domain/models"
	sqlgenerator "analyticDataCenter/analytics-data-center/internal/lib/SQLGenerator"
	"analyticDataCenter/analytics-data-center/internal/storage"
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
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

	queries, err := sqlgenerator.GenerateCountQueries(viewSchema, log.Logger)
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
			SchemaName:    query.SchemaName,
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
	log := a.log.With(slog.String("op", op))

	var (
		tempTbl  []string
		tempMeta []models.TempTable
		wg       sync.WaitGroup
		hasError bool
		mu       sync.Mutex
	)

	// максимум 3 горутины одновременно
	const maxConcurrentWorkers = 3
	sem := make(chan struct{}, maxConcurrentWorkers)

	// размер чанка (универсально для Postgres/ClickHouse)
	const chunkLimit int64 = 500_000

	for _, tempTableInsert := range *countData {
		if tempTableInsert.Count <= 0 {
			continue
		}

		log.Info("запуск вставки и подготовки данных", slog.String("Таблица", tempTableInsert.TableName))
		tempTbl = append(tempTbl, tempTableInsert.TempTableName)
		tempMeta = append(tempMeta, models.TempTable{
			TempTableName: tempTableInsert.TempTableName,
			Source:        tempTableInsert.DataBaseName,
			Schema:        tempTableInsert.SchemaName,
			Table:         tempTableInsert.TableName,
		})

		oltpStorage, err := a.OLTPFactory.GetOLTPStorage(ctx, tempTableInsert.DataBaseName)
		if err != nil {
			log.Error("Невозможно подключиться к OLTP хранилищу", slog.String("error", err.Error()))
			return false, err
		}

		tableName := tempTableInsert.TableName
		sourceName := tempTableInsert.DataBaseName
		tempTableName := tempTableInsert.TempTableName
		total := tempTableInsert.Count

		// идём по таблице батчами по 500к (или меньше на хвосте)
		for start := int64(0); start < total; start += chunkLimit {
			end := start + chunkLimit
			if end > total {
				end = total
			}
			limit := end - start
			if limit <= 0 {
				continue
			}

			wg.Add(1)
			sem <- struct{}{} // занять слот

			go func(start, limit int64, tableName, sourceName, tempTableName string, oltpStorage storage.OLTPDB) {
				defer wg.Done()
				defer func() { <-sem }() // освободить слот

				log.Info("Горутина запущена",
					slog.String("Для таблицы", tableName),
					slog.Int64("offset", start),
					slog.Int64("limit", limit),
				)

				query, err := sqlgenerator.GenerateSelectInsertDataQuery(*viewSchema, start, start+limit, tableName, log.Logger, a.OLTPDbName)
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

				log.Info("получены данные", slog.Int("rows", len(insertData)))

				queryInsert, err := sqlgenerator.GenerateInsertDataQuery(*viewSchema, insertData, tempTableName, log.Logger, a.DWHDbName)
				if err != nil {
					log.Error("ошибка при генерации INSERT запроса", slog.String("error", err.Error()))
					mu.Lock()
					hasError = true
					mu.Unlock()
					return
				}

				err = a.DWHProvider.InsertDataToDWH(ctx, queryInsert.Query)
				if err != nil {
					log.Error("ошибка при вставке данных в таблицу", slog.String("error", err.Error()))
					mu.Lock()
					hasError = true
					mu.Unlock()
					return
				}

				log.Info("Вставка данных завершена",
					slog.String("таблица", tempTableName),
					slog.Int64("offset", start),
					slog.Int64("limit", limit),
				)
			}(start, limit, tableName, sourceName, tempTableName, oltpStorage)
		}
	}

	wg.Wait()

	if hasError {
		_ = a.DeleteTempTables(ctx, tempTbl)
		return false, fmt.Errorf("одна или несколько горутин завершились с ошибкой")
	}

	viewJoin, err := a.prepareViewJoin(ctx, tempMeta, "public")
	if err != nil {
		_ = a.DeleteTempTables(ctx, tempTbl)
		log.Error("Ошибка", slog.String("error", err.Error()))
		return false, err
	}

	query, err := sqlgenerator.CreateViewQuery(*viewSchema, *viewJoin, log.Logger, a.DWHDbName)
	if err != nil {
		_ = a.DeleteTempTables(ctx, tempTbl)
		log.Error("Ошибка", slog.String("error", err.Error()))
		return false, err
	}
	log.Info("Запрос на мердж", slog.String("Запрос", query.Query))
	a.DWHProvider.MergeTempTables(ctx, query.Query)
	_ = a.DeleteTempTables(ctx, tempTbl)
	return true, nil
}

func (a *AnalyticsDataCenterService) calculateWorkerCount(totalCount int64) int64 {
	switch {
	case totalCount > 30_000000:
		return 16
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

func (a *AnalyticsDataCenterService) prepareViewJoin(ctx context.Context, tempTbl []models.TempTable, schemaName string) (viewJoins *models.ViewJoinTable, err error) {
	const op = "analytics.prepareViewJoin"
	log := a.log.With(
		slog.String("op", op),
	)

	var tablesTemp []models.TempTable

	for _, tempTable := range tempTbl {
		if a.DWHDbName == DbClickhouse {
			u, err := url.Parse(a.DWHDbPath)
			if err != nil {
				log.Error("Невозможно распарсить URL", slog.String("error", err.Error()))
				return nil, err
			}
			dbName := strings.Trim(u.Path, "/")
			schemaName = dbName
		}
		columnsTempTables, err := a.DWHProvider.GetColumnsTables(ctx, schemaName, tempTable.TempTableName)
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

		tempTables := tempTable
		tempTables.TempColumns = columnsTemp
		tablesTemp = append(tablesTemp, tempTables)
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

func (a *AnalyticsDataCenterService) transferIndixesAndConstraint(ctx context.Context, viewSchema *models.View, dbName string) error {
	const op = "analytics.transferIndicesAndConstraint"
	log := a.log.With(
		slog.String("op", op),
	)
	if dbName == DbClickhouse {
		return nil
	}
	var OLTPSourceName []string
	var indexTransfers IndexTransfers
	for _, src := range viewSchema.Sources {
		storage, err := a.OLTPFactory.GetOLTPStorage(ctx, src.Name)
		if err != nil {
			log.Error("Невозможно получить хранилище OLTP", slog.String("error", err.Error()))
			return err
		}
		OLTPSourceName = append(OLTPSourceName, src.Name)
		var indexTransfer *IndexTransfer
		var columnIndexes []string
		for _, sch := range src.Schemas {
			for _, tbl := range sch.Tables {
				for _, clmn := range sch.Tables {
					columnIndexes = append(columnIndexes, clmn.Name)
					indexTransfer = &IndexTransfer{
						IndexTransfer: models.IndexTransferTable{
							TableName:  tbl.Name,
							SourceName: src.Name,
							SchemaName: sch.Name,
							Columns:    columnIndexes,
						},
						storage: storage,
					}

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
			query, err := sqlgenerator.TransformIndexDefToSQLExpression(index, transferTable.IndexTransfer.SchemaName, strings.ToLower(transferTable.IndexTransfer.TableName), "public", viewSchema.Name, a.log.Logger)
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
