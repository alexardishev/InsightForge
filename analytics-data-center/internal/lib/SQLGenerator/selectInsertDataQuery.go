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
					columns = append(columns, column.Name)

				}
				// Generate query string
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
