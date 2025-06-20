package sqlgenerator

import (
	"analyticDataCenter/analytics-data-center/internal/domain/models"
	"errors"
	"fmt"
	"log/slog"
	"strings"
)

func GenerateSelectInsertDataQueryPostgres(
	view models.View,
	start int64,
	end int64,
	tableName string,
	logger *slog.Logger,
) (models.Query, error) {
	const op = "sqlgenerator.GenerateSelectInsertDataQueryPostgres"
	logger = logger.With(slog.String("op", op))
	logger.Info("start operation")

	var b strings.Builder
	var primaryColumn string

	for _, source := range view.Sources {
		for _, sch := range source.Schemas {
			for _, tbl := range sch.Tables {
				if tbl.Name != tableName {
					continue
				}

				used := make(map[string]bool)
				columns := make([]string, 0, len(tbl.Columns))

				for _, column := range tbl.Columns {
					if used[column.Name] {
						logger.Error("повторяющееся имя колонки", slog.String("column", column.Name))
						return models.Query{}, errors.New("колонки с одинаковыми именами недопустимы")
					}
					used[column.Name] = true

					// JSON трансформация (Postgres)
					if column.Transform != nil && column.Transform.Type == transformTypeJSON && column.Transform.Mapping != nil {
						for _, mapping := range column.Transform.Mapping.MappingJSON {
							for jsonField, outputColumn := range mapping.Mapping {
								extracted := fmt.Sprintf("%s->>'%s' AS %s", column.Name, jsonField, outputColumn)
								columns = append(columns, extracted)
							}
						}
						columns = append(columns, column.Name)
					} else {
						columns = append(columns, column.Name)
					}

					if column.Transform != nil && column.Transform.Type == transformTypeFieldTransform && column.Transform.Mapping != nil {
						mapping := column.Transform.Mapping
						var a strings.Builder
						a.WriteString("CASE ")
						for key, value := range mapping.Mapping {
							safeKey := strings.ReplaceAll(key, "'", "''")
							safeValue := strings.ReplaceAll(value, "'", "''")
							a.WriteString(fmt.Sprintf("WHEN %s = '%s' THEN '%s' ", column.Name, safeKey, safeValue))
						}
						alias := mapping.AliasNewColumnTransform
						if alias == "" {
							alias = column.Name + "_transformed"
						}
						a.WriteString(fmt.Sprintf("END as %s", alias))
						columns = append(columns, a.String())
					}

					if column.IsPrimaryKey && primaryColumn == "" {
						primaryColumn = column.Name
					}
				}

				if primaryColumn != "" {
					b.WriteString(fmt.Sprintf("SELECT %s FROM %s ORDER BY %s OFFSET %d LIMIT %d",
						strings.Join(columns, ", "),
						tableName,
						primaryColumn,
						start,
						end-start,
					))
				} else {
					b.WriteString(fmt.Sprintf("SELECT %s FROM %s OFFSET %d LIMIT %d",
						strings.Join(columns, ", "),
						tableName,
						start,
						end-start,
					))
				}

				return models.Query{
					TableName:  tbl.Name,
					SourceName: source.Name,
					Query:      b.String(),
				}, nil
			}
		}
	}
	logger.Info("таблица не найдена в представлении", slog.String("таблица", tableName))
	return models.Query{}, fmt.Errorf("таблица %s не найдена в представлении", tableName)
}
