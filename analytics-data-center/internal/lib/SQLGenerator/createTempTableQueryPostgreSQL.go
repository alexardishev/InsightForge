package sqlgenerator

import (
	"analyticDataCenter/analytics-data-center/internal/domain/models"
	"analyticDataCenter/analytics-data-center/internal/lib/duplicate"
	"fmt"
	"log/slog"
	"strings"
	"unicode/utf8"

	"github.com/lib/pq"
)

func MapTypeToPostgres(typ string) string {
	original := strings.TrimSpace(typ)
	typeLower := strings.ToLower(original)
	isArray := false

	if strings.HasPrefix(typeLower, "_") {
		isArray = true
		typeLower = strings.TrimPrefix(typeLower, "_")
	}
	if strings.HasSuffix(typeLower, "[]") {
		isArray = true
		typeLower = strings.TrimSuffix(typeLower, "[]")
	}

	baseType := typeLower
	length := ""
	if idx := strings.Index(baseType, "("); idx != -1 && strings.HasSuffix(baseType, ")") {
		length = strings.TrimSuffix(baseType[idx+1:], ")")
		baseType = strings.TrimSpace(baseType[:idx])
	}

	var mapped string
	switch baseType {
	case "int", "integer", "int4":
		mapped = "INTEGER"
	case "bigint", "int8":
		mapped = "BIGINT"
	case "float", "float4", "real":
		mapped = "REAL"
	case "double precision", "float8":
		mapped = "DOUBLE PRECISION"
	case "bool", "boolean":
		mapped = "BOOLEAN"
	case "smallint", "int2":
		mapped = "SMALLINT"
	case "date":
		mapped = "DATE"
	case "timestamp", "timestamp without time zone":
		mapped = "TIMESTAMP"
	case "timestamp with time zone", "timestamptz":
		mapped = "TIMESTAMPTZ"
	case "time", "time without time zone":
		mapped = "TIME"
	case "timetz", "time with time zone":
		mapped = "TIMETZ"
	case "interval":
		mapped = "INTERVAL"
	case "numeric", "decimal":
		mapped = "NUMERIC"
	case "money":
		mapped = "MONEY"
	case "json", "jsonb":
		mapped = "JSONB"
	case "uuid":
		mapped = "UUID"
	case "inet":
		mapped = "INET"
	case "cidr":
		mapped = "CIDR"
	case "macaddr":
		mapped = "MACADDR"
	case "macaddr8":
		mapped = "MACADDR8"
	case "bytea":
		mapped = "BYTEA"
	case "bit":
		mapped = "VARBIT"
	case "varbit", "bit varying":
		mapped = "VARBIT"
	case "xml":
		mapped = "XML"
	case "text":
		mapped = "TEXT"
	case "varchar", "character varying":
		mapped = "VARCHAR"
	case "character", "char":
		if length == "" {
			mapped = "TEXT"
			break
		}
		mapped = "CHAR"
	case "array":
		mapped = "TEXT[]"
	default:
		mapped = "TEXT"
	}
	if strings.EqualFold(baseType, "user-defined") {
		mapped = "TEXT"
	}
	if strings.EqualFold(mapped, "user-defined") {
		if original != "" && !strings.EqualFold(original, "user-defined") {
			mapped = original
		} else {
			mapped = "TEXT"
		}
	}

	if length != "" {
		mapped = fmt.Sprintf("%s(%s)", mapped, length)
	}

	if isArray && mapped != "TEXT[]" {
		return mapped + "[]"
	}
	return mapped
}

func GenerateQueryCreateTempTablePostgres(
	schema *models.View,
	logger *slog.Logger,
	_ string,
) (models.Queries, []string, error) {
	const op = "sqlgenerator.GenerateQueryCreateTempTablePostgres"
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
				quotedTable := pq.QuoteIdentifier(tableName)
				_, err := b.WriteString(fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n", quotedTable))
				if err != nil {
					logger.Error("ошибка", slog.String("error", err.Error()))
					return models.Queries{}, nil, err
				}

				linePrimary := ""

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
					quotedCol := pq.QuoteIdentifier(colName)
					colType := MapColumnTypeToPostgresDDL(col)
					isNotNull := "NOT NULL"
					var line string
					if !col.IsNullable {
						line = fmt.Sprintf("  %s %s %s", quotedCol, colType, isNotNull)
					} else {
						line = fmt.Sprintf("  %s %s", quotedCol, colType)
					}
					if col.IsPrimaryKey {
						constraintName := pq.QuoteIdentifier(shortenIdentifier(fmt.Sprintf("%s_%s_prk", colName, tableName)))
						linePrimary = fmt.Sprintf(", CONSTRAINT %s PRIMARY KEY (%s)", constraintName, quotedCol)
					}
					if idx < len(cleanList)-1 {
						line += ","
					}
					_, err = b.WriteString(line + "\n")
					if err != nil {
						logger.Error("ошибка", slog.String("error", err.Error()))
						return models.Queries{}, nil, err
					}
				}
				_, err = b.WriteString(fmt.Sprintf(" %s );\n", linePrimary))
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

func shortenIdentifier(name string) string {
	const maxBytes = 63
	if len(name) <= maxBytes {
		return name
	}
	for i := range name {
		if i > maxBytes {
			return name[:i]
		}
	}
	return name
}
