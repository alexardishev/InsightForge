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

func GenerateInsertDataQueryPostgres(view models.View, selectData []map[string]interface{}, tempTableName string, logger *slog.Logger) (models.Query, error) {
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

	logger.Info("ДАТА ДЛЯ ВСТАВКИ", slog.Any("Нулевой элемент", selectData[0]))

	columnNames := make(map[string]struct{})
	for _, src := range view.Sources {
		for _, sch := range src.Schemas {
			for _, tbl := range sch.Tables {
				for _, clmn := range tbl.Columns {
					if clmn.Transform != nil && clmn.Transform.Type == transformTypeJSON && clmn.Transform.Mapping != nil {
						logger.Info("Начинаю работу с JSON трансформацией")
						for _, mapping := range clmn.Transform.Mapping.MappingJSON {
							for _, outputCol := range mapping.Mapping {
								columnNames[outputCol] = struct{}{}
							}
						}
						columnNames[clmn.Name] = struct{}{}
					} else {
						columnNames[clmn.Name] = struct{}{}
					}
					if clmn.Transform != nil && clmn.Transform.Type == transformTypeFieldTransform && clmn.Transform.Mapping != nil {
						logger.Info("Начинаю работу с transformTypeFieldTransform трансформацией")
						mapping := clmn.Transform.Mapping
						columnNames[mapping.AliasNewColumnTransform] = struct{}{}
						columnNames[clmn.Name] = struct{}{}
					} else {
						columnNames[clmn.Name] = struct{}{}
					}
				}
			}
		}
	}

	for colName := range columnNames {
		if _, ok := availableColumns[colName]; ok {
			logger.Info("Колонка добавлена", slog.String("name", colName))
			columns = append(columns, colName)
		}
	}

	if len(columns) == 0 {
		logger.Error("Не найдены колонки для вставки")
		return models.Query{}, errors.New("не найдены подходящие колонки для вставки")
	}

	for _, row := range selectData {
		var valueStrings []string
		for _, col := range columns {
			val, ok := row[col]
			if !ok || val == nil {
				valueStrings = append(valueStrings, "NULL")
				continue
			}

			switch v := val.(type) {
			case string:
				safe := strings.ReplaceAll(v, "'", "''")
				valueStrings = append(valueStrings, fmt.Sprintf("'%s'", safe))
			case []byte:
				safe := strings.ReplaceAll(string(v), "'", "''")
				valueStrings = append(valueStrings, fmt.Sprintf("'%s'", safe))
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

	b.WriteString(fmt.Sprintf("INSERT INTO %s (%s) VALUES %s",
		tempTableName,
		strings.Join(columns, ", "),
		strings.Join(valuesInsertData, ", ")))

	finalQuery := b.String()
	logger.Info("Сгенерированный SQL", slog.String("query", finalQuery))

	return models.Query{
		Query:     finalQuery,
		TableName: tempTableName,
	}, nil
}
