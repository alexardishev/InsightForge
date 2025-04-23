package sqlgenerator

import (
	"analyticDataCenter/analytics-data-center/internal/domain/models"
	"analyticDataCenter/analytics-data-center/internal/lib/duplicate"
	"fmt"
	"log/slog"
	"strings"
)

func GenerateQueryCreateTempTablePostgres(schema *models.View, logger *slog.Logger) (queriesCreateTable models.Queries, duplicateNamesTable []string, err error) {
	const op = "sqlgenerator.GenerateQueryCreateTempTable"
	logger = logger.With(slog.String("op", op))
	logger.Info("start operation")
	var duplicateColumnNames []string
	var queryObject []models.Query

	for _, source := range schema.Sources {
		logger.Info("БД", slog.String("database", source.Name))
		for _, sch := range source.Schemas {
			logger.Info("Schemas", slog.String("Schemas", sch.Name))

			for _, tbl := range sch.Tables {
				logger.Info("Tables", slog.String("Tables", tbl.Name))

				var b strings.Builder

				// Название таблицы: temp_{source}_{schema}_{table}
				tableName := fmt.Sprintf("temp_%s_%s_%s", source.Name, sch.Name, tbl.Name)
				_, err := b.WriteString(fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n", tableName))
				if err != nil {
					logger.Error("ошибка", slog.String("error", err.Error()))
					return models.Queries{}, nil, err
				}

				cleanList, duplicateList := duplicate.RemoveDuplicateColumns(tbl.Columns)
				if len(duplicateList) > 0 {
					logger.Warn("duplicate", slog.Any("Дублирующие имена колонок", duplicateList), slog.Any("в таблице", tbl.Name))
					duplicateColumnNames = append(duplicateColumnNames, duplicateList...)
				}

				for idx, col := range cleanList {
					colName := col.Alias
					if colName == "" {
						colName = col.Name
					}
					colType := col.Type
					isNotNull := "NOT NULL"
					if colType == "" {
						logger.Info("не указан тип по умолчанию text", slog.String("не задан тип для колонки", colName))
						colType = "TEXT"
					}
					var line string
					if !col.IsNullable {
						line = fmt.Sprintf("  %s %s %s", colName, colType, isNotNull)
					} else {
						line = fmt.Sprintf("  %s %s", colName, colType)
					}

					if idx < len(tbl.Columns)-1 {
						line += ","
					}

					_, err = b.WriteString(line + "\n")
					if err != nil {
						logger.Error("ошибка", slog.String("error", err.Error()))
						return models.Queries{}, nil, err
					}
				}

				_, err = b.WriteString(");\n")
				if err != nil {
					logger.Error("ошибка", slog.String("error", err.Error()))
					return models.Queries{}, nil, err
				}
				querySt := &models.Query{
					TableName: tableName,
					Query:     b.String(),
				}
				queryObject = append(queryObject, *querySt)

			}
		}
	}

	return models.Queries{
		Queries: queryObject,
	}, duplicateColumnNames, nil
}
