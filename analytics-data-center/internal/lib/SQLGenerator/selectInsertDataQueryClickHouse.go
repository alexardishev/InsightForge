package sqlgenerator

import (
	"analyticDataCenter/analytics-data-center/internal/domain/models"
	"errors"
	"fmt"
	"log/slog"
	"strings"
)

func GenerateSelectInsertDataQueryClickhouse(
	view models.View,
	start int64,
	end int64,
	tableName string,
	logger *slog.Logger,
) (models.Query, error) {
	const op = "sqlgenerator.GenerateSelectInsertDataQueryClickhouse"
	logger = logger.With(slog.String("op", op))
	logger.Info("start operation")

	var b strings.Builder
	var primaryColumn string

	resolveColumnName := func(column models.Column) string {
		if column.Alias != "" {
			return column.Alias
		}
		return column.Name
	}

	for _, source := range view.Sources {
		for _, sch := range source.Schemas {
			for _, tbl := range sch.Tables {
				if tbl.Name != tableName {
					continue
				}

				used := make(map[string]bool)
				columns := make([]string, 0, len(tbl.Columns))

				for _, column := range tbl.Columns {
					targetName := resolveColumnName(column)
					if used[targetName] {
						logger.Error("повторяющееся имя колонки", slog.String("column", targetName))
						return models.Query{}, errors.New("колонки с одинаковыми именами недопустимы")
					}
					used[targetName] = true

					columnExpr := column.Name
					if column.Alias != "" {
						columnExpr = fmt.Sprintf("%s AS %s", column.Name, column.Alias)
					}

					// JSON трансформация (ClickHouse)
					if column.Transform != nil && column.Transform.Type == transformTypeJSON && column.Transform.Mapping != nil {
						for _, mapping := range column.Transform.Mapping.MappingJSON {
							for jsonField, outputColumn := range mapping.Mapping {
								// Пример: JSONExtractString(column, 'jsonField') AS outputColumn
								extracted := fmt.Sprintf("JSONExtractString(%s, '%s') AS %s", column.Name, jsonField, outputColumn)
								columns = append(columns, extracted)
							}
						}
						columns = append(columns, columnExpr)
					} else {
						columns = append(columns, columnExpr)
					}

					// CASE (Field Transform)
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

				// В ClickHouse — LIMIT <limit> OFFSET <offset>
				limit := end - start
				if primaryColumn != "" {
					b.WriteString(fmt.Sprintf("SELECT %s FROM %s ORDER BY %s LIMIT %d OFFSET %d",
						strings.Join(columns, ", "),
						tableName,
						primaryColumn,
						limit,
						start,
					))
				} else {
					b.WriteString(fmt.Sprintf("SELECT %s FROM %s LIMIT %d OFFSET %d",
						strings.Join(columns, ", "),
						tableName,
						limit,
						start,
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
