package sqlgenerator

import (
	"analyticDataCenter/analytics-data-center/internal/domain/models"
	"analyticDataCenter/analytics-data-center/internal/lib/duplicate"
	"fmt"
	"log/slog"
	"strings"
)

// Маппинг типов для ClickHouse
func MapTypeToClickhouse(pgType string) string {
	switch strings.ToLower(pgType) {
	case "text", "varchar", "character varying":
		return "String"
	case "int", "integer", "int4":
		return "Int32"
	case "bigint", "int8":
		return "Int64"
	case "double precision", "float8":
		return "Float64"
	case "float4", "real":
		return "Float32"
	case "boolean", "bool":
		return "UInt8"
	case "date":
		return "Date"
	case "timestamp", "timestamp without time zone", "timestamptz":
		return "DateTime"
	default:
		return "String"
	}
}

func GenerateQueryCreateTempTableClickhouse(
	schema *models.View,
	logger *slog.Logger,
) (models.Queries, []string, error) {
	const op = "sqlgenerator.GenerateQueryCreateTempTableClickhouse"
	logger = logger.With(slog.String("op", op))
	logger.Info("start operation")
	var duplicateColumnNames []string
	var queryObject []models.Query

	for _, source := range schema.Sources {
		for _, sch := range source.Schemas {
			for _, tbl := range sch.Tables {
				logger.Info("Table", slog.String("Table", tbl.Name))
				var b strings.Builder
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
								tbl.Columns = append(tbl.Columns, *column)
							}
						}
					}
					if clmn.Transform.Type == transformTypeFieldTransform {
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
					colType := MapTypeToClickhouse(col.Type)
					if col.IsNullable {
						colType = fmt.Sprintf("Nullable(%s)", colType)
					}
					line := fmt.Sprintf("  %s %s", colName, colType)
					if idx < len(cleanList)-1 {
						line += ","
					}
					_, err = b.WriteString(line + "\n")
					if err != nil {
						logger.Error("ошибка", slog.String("error", err.Error()))
						return models.Queries{}, nil, err
					}
				}
				_, err = b.WriteString(") ENGINE = Memory;\n")
				if err != nil {
					logger.Error("ошибка", slog.String("error", err.Error()))
					return models.Queries{}, nil, err
				}
				querySt := &models.Query{
					TableName: tableName,
					Query:     b.String(),
				}
				fmt.Sprintln(querySt.Query)
				queryObject = append(queryObject, *querySt)
			}
		}
	}

	return models.Queries{
		Queries: queryObject,
	}, duplicateColumnNames, nil
}
