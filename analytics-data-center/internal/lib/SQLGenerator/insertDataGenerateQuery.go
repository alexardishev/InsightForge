package sqlgenerator

import (
	"analyticDataCenter/analytics-data-center/internal/domain/models"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/gofrs/uuid"
)

func GeneratetInsertDataQuery(view models.View, selectData []map[string]interface{}, tempTableName string, logger *slog.Logger) (models.Query, error) {
	const op = "sqlgenerator.GenerateSelectInsertDataQuery"
	logger = logger.With(slog.String("op", op))
	logger.Info("СТАРТ ПОДГОТОВКИ ЗАПРОСА")

	if len(selectData) == 0 {
		logger.Info("ДАННЫЕ ПУСТЫЕ")
		return models.Query{}, errors.New("пустой набор данных для вставки")
	}

	var b strings.Builder
	var columns []string
	var valuesInsertData []string

	availableColumns := make(map[string]struct{})
	for col := range selectData[0] {
		availableColumns[col] = struct{}{}
	}

	for _, src := range view.Sources {
		for _, sch := range src.Schemas {
			for _, tbl := range sch.Tables {
				for _, clmn := range tbl.Columns {
					if _, ok := availableColumns[clmn.Name]; ok {
						logger.Info("Колонка добавлена", slog.String("name", clmn.Name))
						columns = append(columns, clmn.Name)
					}
				}
			}
		}
	}

	if len(columns) == 0 {
		logger.Error("Не найдены колонки для вставки")
		return models.Query{}, errors.New("не найдены подходящие колонки для вставки")
	}

	testSelectData := selectData[:2]
	for _, row := range testSelectData {
		var valueStrings []string
		for _, col := range columns {
			val, ok := row[col]
			if !ok || val == nil {
				valueStrings = append(valueStrings, "NULL")
				continue
			}

			switch v := val.(type) {
			case string:
				valueStrings = append(valueStrings, fmt.Sprintf("'%s'", v))
			case []byte:
				valueStrings = append(valueStrings, fmt.Sprintf("'%s'", string(v)))
			case int, int64, float64:
				valueStrings = append(valueStrings, fmt.Sprintf("%v", v))
			case uuid.UUID:
				valueStrings = append(valueStrings, fmt.Sprintf("'%s'", v.String()))
			case bool:
				if v {
					valueStrings = append(valueStrings, "TRUE")
				} else {
					valueStrings = append(valueStrings, "FALSE")
				}
			case time.Time:
				valueStrings = append(valueStrings, fmt.Sprintf("'%s'", v.Format("2006-01-02 15:04:05")))
			default:
				return models.Query{}, fmt.Errorf("неподдерживаемый тип значения для колонки %s (%T)", col, v)
			}
		}
		wrapped := fmt.Sprintf("(%s)", strings.Join(valueStrings, ", "))
		valuesInsertData = append(valuesInsertData, wrapped)
	}

	// Финальный SQL
	b.WriteString(fmt.Sprintf("INSERT INTO %s (%s) VALUES %s",
		tempTableName,
		strings.Join(columns, ", "),
		strings.Join(valuesInsertData, ", ")))

	finalQuery := b.String()

	return models.Query{
		Query:     finalQuery,
		TableName: tempTableName,
	}, nil
}
