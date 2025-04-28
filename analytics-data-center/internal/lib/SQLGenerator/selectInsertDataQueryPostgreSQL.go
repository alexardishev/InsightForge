package sqlgenerator

import (
	"analyticDataCenter/analytics-data-center/internal/domain/models"
	"errors"
	"fmt"
	"log/slog"
	"strings"
)

func GenerateSelectInsertDataQuery(view models.View, start int64, end int64, tableName string, logger *slog.Logger) (query models.Query, err error) {
	const op = "sqlgenerator.GenerateSelectInsertDataQuery"
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

					// Если колонка с JSON-трансформацией
					if column.Transform != nil && column.Transform.Type == "JSON" && column.Transform.Mapping != nil {
						logger.Info("Обработка трансформации JSON", slog.String("column", column.Name))

						for _, mapping := range column.Transform.Mapping.MappingJSON {
							for jsonField, outputColumn := range mapping.Mapping {
								extracted := fmt.Sprintf("%s->>'%s' AS %s", column.Name, jsonField, outputColumn)
								columns = append(columns, extracted)
							}
						}
						columns = append(columns, column.Name)
					} else {
						// Просто обычная колонка
						columns = append(columns, column.Name)
					}

					if column.IsPrimaryKey && primaryColumn == "" {
						primaryColumn = column.Name
					}
				}

				// Собираем финальный SELECT
				if primaryColumn != "" {
					_, err := b.WriteString(fmt.Sprintf("SELECT %s FROM %s ORDER BY %s OFFSET %d LIMIT %d",
						strings.Join(columns, ", "),
						tableName,
						primaryColumn,
						start,
						end-start,
					))
					if err != nil {
						logger.Error("ошибка", slog.String("error", err.Error()))
						return models.Query{}, err
					}
				} else {
					_, err := b.WriteString(fmt.Sprintf("SELECT %s FROM %s OFFSET %d LIMIT %d",
						strings.Join(columns, ", "),
						tableName,
						start,
						end-start,
					))
					if err != nil {
						logger.Error("ошибка", slog.String("error", err.Error()))
						return models.Query{}, err
					}
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
