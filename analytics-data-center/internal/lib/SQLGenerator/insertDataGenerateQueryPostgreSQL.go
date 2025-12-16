package sqlgenerator

import (
	"analyticDataCenter/analytics-data-center/internal/domain/models"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"encoding/hex"

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
	resolveColumnName := func(col models.Column) string {
		if col.Alias != "" {
			return col.Alias
		}
		return col.Name
	}

	for _, src := range view.Sources {
		for _, sch := range src.Schemas {
			for _, tbl := range sch.Tables {
				for _, clmn := range tbl.Columns {
					finalName := resolveColumnName(clmn)
					if clmn.Transform != nil && clmn.Transform.Type == transformTypeJSON && clmn.Transform.Mapping != nil {
						logger.Info("Начинаю работу с JSON трансформацией")
						for _, mapping := range clmn.Transform.Mapping.MappingJSON {
							for _, outputCol := range mapping.Mapping {
								columnNames[outputCol] = struct{}{}
							}
						}
						columnNames[finalName] = struct{}{}
					} else {
						columnNames[finalName] = struct{}{}
					}
					if clmn.Transform != nil && clmn.Transform.Type == transformTypeFieldTransform && clmn.Transform.Mapping != nil {
						logger.Info("Начинаю работу с transformTypeFieldTransform трансформацией")
						mapping := clmn.Transform.Mapping
						aliasName := mapping.AliasNewColumnTransform
						if aliasName == "" {
							aliasName = clmn.Name + "_transformed"
						}
						columnNames[aliasName] = struct{}{}
						columnNames[finalName] = struct{}{}
					} else {
						columnNames[finalName] = struct{}{}
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
	sort.Strings(columns)

	if len(columns) == 0 {
		logger.Error("Не найдены колонки для вставки")
		return models.Query{}, errors.New("не найдены подходящие колонки для вставки")
	}

	columnTypes := buildColumnTypeMap(view)

	for _, row := range selectData {
		var valueStrings []string
		for _, col := range columns {
			val, ok := row[col]
			if !ok || val == nil {
				valueStrings = append(valueStrings, "NULL")
				continue
			}

			typeLower := strings.ToLower(columnTypes[col])
			switch v := val.(type) {
			case string:
				if isBitType(typeLower) {
					valueStrings = append(valueStrings, formatBitLiteral(v))
				} else {
					safe := strings.ReplaceAll(v, "'", "''")
					valueStrings = append(valueStrings, fmt.Sprintf("'%s'", safe))
				}
			case []byte:
				switch {
				case isByteaType(typeLower):
					valueStrings = append(valueStrings, formatPostgresBytea(v))
				case isBitType(typeLower):
					valueStrings = append(valueStrings, formatBitLiteral(string(v)))
				default:
					safe := strings.ReplaceAll(string(v), "'", "''")
					valueStrings = append(valueStrings, fmt.Sprintf("'%s'", safe))
				}
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
				valueStrings = append(valueStrings, formatTimeLiteral(v, typeLower))
			case []interface{}:
				valueStrings = append(valueStrings, formatPostgresArray(v, typeLower))
			case []int:
				arr := make([]interface{}, len(v))
				for i, item := range v {
					arr[i] = item
				}
				valueStrings = append(valueStrings, formatPostgresArray(arr, typeLower))
			case []int64:
				arr := make([]interface{}, len(v))
				for i, item := range v {
					arr[i] = item
				}
				valueStrings = append(valueStrings, formatPostgresArray(arr, typeLower))
			case []float64:
				arr := make([]interface{}, len(v))
				for i, item := range v {
					arr[i] = item
				}
				valueStrings = append(valueStrings, formatPostgresArray(arr, typeLower))
			case []string:
				arr := make([]interface{}, len(v))
				for i, item := range v {
					arr[i] = item
				}
				valueStrings = append(valueStrings, formatPostgresArray(arr, typeLower))
			case []uuid.UUID:
				arr := make([]interface{}, len(v))
				for i, item := range v {
					arr[i] = item.String()
				}
				valueStrings = append(valueStrings, formatPostgresArray(arr, typeLower))
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

func formatPostgresArray(items []interface{}, typeLower string) string {
	parts := make([]string, 0, len(items))
	for _, it := range items {
		switch v := it.(type) {
		case nil:
			parts = append(parts, "NULL")
		case string:
			if isBitType(typeLower) {
				parts = append(parts, formatBitLiteral(v))
			} else {
				safe := strings.ReplaceAll(v, "\"", "\\\"")
				safe = strings.ReplaceAll(safe, "'", "''")
				parts = append(parts, fmt.Sprintf("\"%s\"", safe))
			}
		case []byte:
			switch {
			case isByteaType(typeLower):
				parts = append(parts, formatPostgresBytea(v))
			case isBitType(typeLower):
				parts = append(parts, formatBitLiteral(string(v)))
			default:
				safe := strings.ReplaceAll(string(v), "'", "''")
				parts = append(parts, fmt.Sprintf("'%s'", safe))
			}
		default:
			parts = append(parts, fmt.Sprintf("%v", v))
		}
	}
	return fmt.Sprintf("'{%s}'", strings.Join(parts, ","))
}

func formatPostgresBytea(b []byte) string {
	return fmt.Sprintf("E'\\\\x%s'", hex.EncodeToString(b))
}

func formatBitLiteral(raw string) string {
	var b strings.Builder
	for _, r := range raw {
		if r == '0' || r == '1' {
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		safe := strings.ReplaceAll(raw, "'", "''")
		return fmt.Sprintf("'%s'", safe)
	}
	return fmt.Sprintf("B'%s'", b.String())
}

func buildColumnTypeMap(view models.View) map[string]string {
	res := make(map[string]string)
	resolveColumnName := func(col models.Column) string {
		if col.Alias != "" {
			return col.Alias
		}
		return col.Name
	}
	for _, src := range view.Sources {
		for _, sch := range src.Schemas {
			for _, tbl := range sch.Tables {
				for _, col := range tbl.Columns {
					res[resolveColumnName(col)] = col.Type
				}
			}
		}
	}
	return res
}

func isBitType(typeLower string) bool {
	return strings.HasPrefix(typeLower, "bit") || strings.HasPrefix(typeLower, "varbit")
}

func isByteaType(typeLower string) bool {
	return strings.Contains(typeLower, "bytea")
}

func formatTimeLiteral(t time.Time, typeLower string) string {
	switch {
	case strings.Contains(typeLower, "date"):
		return fmt.Sprintf("'%s'", t.Format("2006-01-02"))
	case strings.Contains(typeLower, "timetz"):
		return fmt.Sprintf("'%s'", t.Format("15:04:05-07"))
	case strings.Contains(typeLower, "time"):
		return fmt.Sprintf("'%s'", t.Format("15:04:05"))
	case strings.Contains(typeLower, "timestamptz"):
		return fmt.Sprintf("'%s'", t.Format("2006-01-02 15:04:05-07"))
	default: // timestamp without tz
		return fmt.Sprintf("'%s'", t.Format("2006-01-02 15:04:05"))
	}
}
