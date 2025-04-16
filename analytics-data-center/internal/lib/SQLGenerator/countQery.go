package sqlgenerator

import (
	"analyticDataCenter/analytics-data-center/internal/domain/models"
	"fmt"
	"log/slog"
	"strings"
)

func GenerateCountQueries(view models.View, logger *slog.Logger) (queriesCreateTable models.Queries, err error) {
	const op = "sqlgenerator.CountQueryInsertData"
	logger = logger.With(slog.String("op", op))
	logger.Info("start operation")
	var queryObject []models.Query

	for _, source := range view.Sources {
		logger.Info("БД", slog.String("database", source.Name))
		for _, sch := range source.Schemas {
			logger.Info("Schemas", slog.String("Schemas", sch.Name))

			for _, tbl := range sch.Tables {
				logger.Info("Tables", slog.String("Tables", tbl.Name))

				var b strings.Builder

				_, err := b.WriteString(fmt.Sprintf("SELECT COUNT(*) FROM %s", tbl.Name))
				if err != nil {
					logger.Error("ошибка", slog.String("error", err.Error()))
					return models.Queries{}, err
				}
				queryCount := models.Query{
					TableName:  tbl.Name,
					Query:      b.String(),
					SourceName: source.Name,
				}

				queryObject = append(queryObject, queryCount)

			}
		}
	}
	return models.Queries{Queries: queryObject}, nil

}
