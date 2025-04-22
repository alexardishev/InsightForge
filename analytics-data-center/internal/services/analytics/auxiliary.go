package serviceanalytics

import (
	"analyticDataCenter/analytics-data-center/internal/domain/models"
	sqlgenerator "analyticDataCenter/analytics-data-center/internal/lib/SQLGenerator"
	"context"
	"log/slog"
	"runtime"
)

func (a *AnalyticsDataCenterService) createTempTables(ctx context.Context, quries models.Queries) error {
	const op = "analytics.createTempTables"
	var errorCreate error
	log := a.log.With(
		slog.String("op", op),
	)

	for _, query := range quries.Queries {
		err := a.TableProvider.CreateTempTable(ctx, query.Query, query.TableName)
		if err != nil {
			errorCreate = err
			log.Error("не удалось создать временные таблицы", slog.String("error", err.Error()))
			break

		}

	}
	if errorCreate != nil {
		for _, tableQuery := range quries.Queries {
			err := a.TableProvider.DeleteTempTable(ctx, tableQuery.TableName)
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

func (a *AnalyticsDataCenterService) getCountInsertData(ctx context.Context, viewSchema models.View) ([]models.CountInsertData, error) {
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

	for _, query := range queries.Queries {
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
			TableName:    query.TableName,
			Count:        count,
			DataBaseName: query.SourceName,
		}

		sliceCountInsertData = append(sliceCountInsertData, item)

	}
	log.Info("готово", slog.Int("tablesWithData", len(sliceCountInsertData)))
	return sliceCountInsertData, nil
}

func (a *AnalyticsDataCenterService) prepareDataForInsert(ctx context.Context, countData *[]models.CountInsertData, viewSchema *models.View) (bool, error) {
	const op = "analytics.prepareDataForInsert"
	log := a.log.With(
		slog.String("op", op),
	)
	log.Info("запуск вставки данных")

	for _, tempTableInsert := range *countData {
		oltpStorage, err := a.OLTPFactory.GetOLTPStorage(ctx, tempTableInsert.DataBaseName)
		if err != nil {
			log.Error("Невозможно подключиться к OLTP хранилищу", slog.String("error", err.Error()))
			return false, err
		}

		procCount := int64(runtime.GOMAXPROCS(2))
		chunkNum := tempTableInsert.Count/int64(procCount) + 1

		go func() { // ← Открыта анонимная функция

			for i := int64(0); i < chunkNum; i++ {
				start := i * tempTableInsert.Count / procCount
				end := (i + 1) * tempTableInsert.Count / procCount

				query, err := sqlgenerator.GenerateSelectInsertDataQuery(*viewSchema, start, end, tempTableInsert.TableName, log)
				insertData, err := oltpStorage.SelectDataToInsert(ctx, "") // <- query не используется

			}
		}()
	}

	return true, nil
}
