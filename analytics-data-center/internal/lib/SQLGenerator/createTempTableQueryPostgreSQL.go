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
	var line_primary string
	for _, source := range schema.Sources {
		for _, sch := range source.Schemas {
			for _, tbl := range sch.Tables {
				logger.Info("Tables", slog.String("Tables", tbl.Name))

				var b strings.Builder

				// Название таблицы: temp_{source}_{schema}_{table}
				logger.Info("Tables", slog.String("Tables", tbl.Name))

				tableName := fmt.Sprintf("temp_%s_%s_%s", source.Name, sch.Name, tbl.Name)
				_, err := b.WriteString(fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n", tableName))
				if err != nil {
					logger.Error("ошибка", slog.String("error", err.Error()))
					return models.Queries{}, nil, err
				}

				for _, clmn := range tbl.Columns {
					if clmn.Transform == nil {
						continue
					}
					if clmn.Transform.Type == transformTypeJSON {
						logger.Info("Начинаю работу с JSON трансформацией")
						mapping := clmn.Transform.Mapping.MappingJSON
						for _, colmnMappingRow := range mapping {
							for _, value := range colmnMappingRow.Mapping {

								colName := value
								typeClm := colmnMappingRow.TypeField
								column := &models.Column{
									Name:       colName,
									Type:       typeClm,
									IsNullable: true,
								}
								logger.Info("Колонка найдена для трансформации",
									slog.String("name", column.Name),
									slog.String("type", column.Type),
									slog.Bool("is_nullable", column.IsNullable),
								)

								tbl.Columns = append(tbl.Columns, *column)
							}

						}

					}

					if clmn.Transform.Type == transformTypeFieldTransform {
						logger.Info("Начинаю работу с transformTypeFieldTransform трансформацией")
						mapping := clmn.Transform.Mapping
						colName := mapping.AliasNewColumnTransform
						column := &models.Column{
							Name:       colName,
							IsNullable: true,
						}
						tbl.Columns = append(tbl.Columns, *column)
					}
				}

				cleanList, duplicateList := duplicate.RemoveDuplicateColumns(tbl.Columns)
				if len(duplicateList) > 0 {
					logger.Warn("duplicate", slog.Any("Дублирующие имена колонок", duplicateList), slog.Any("в таблице", tbl.Name))
					duplicateColumnNames = append(duplicateColumnNames, duplicateList...)
				}
				logger.Info("Массив Колонок", slog.Any("Колонки", cleanList))
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

					if col.IsPrimaryKey {
						line_primary = fmt.Sprintf(" , CONSTRAINT %s_%s_prk PRIMARY KEY (%s)", colName, tableName, colName)
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

				_, err = b.WriteString(fmt.Sprintf(" %s );\n", line_primary))
				if err != nil {
					logger.Error("ошибка", slog.String("error", err.Error()))
					return models.Queries{}, nil, err
				}
				line_primary = ""
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
