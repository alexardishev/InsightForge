package sqlgenerator

import (
	"analyticDataCenter/analytics-data-center/internal/domain/models"
	"fmt"
	"strings"
)

func MapColumnTypeToPostgresDDL(col models.Column) string {
	dataType := strings.ToLower(strings.TrimSpace(col.DataType))
	udtName := strings.ToLower(strings.TrimSpace(col.UdtName))
	baseType := dataType
	isArray := false

	if udtName == "" && col.Type != "" {
		udtName = strings.ToLower(strings.TrimSpace(col.Type))
	}

	if strings.HasPrefix(udtName, "_") {
		isArray = true
		udtName = strings.TrimPrefix(udtName, "_")
	}

	if baseType == "" {
		baseType = udtName
	}

	if strings.EqualFold(baseType, "array") && udtName != "" {
		baseType = udtName
		isArray = true
	}

	isUserDefined := strings.EqualFold(baseType, "user-defined") ||
		(!strings.EqualFold(col.UdtSchema, "") &&
			!strings.EqualFold(col.UdtSchema, "pg_catalog") &&
			!strings.EqualFold(col.UdtSchema, "information_schema") &&
			udtName != "")

	if isUserDefined {
		if isArray {
			return "TEXT[]"
		}
		return "TEXT"
	}

	mapped := mapBasePostgresType(baseType, col)

	if isArray {
		return mapped + "[]"
	}
	return mapped
}

func mapBasePostgresType(baseType string, col models.Column) string {
	switch baseType {
	case "smallint", "int2":
		return "SMALLINT"
	case "int", "integer", "int4":
		return "INTEGER"
	case "bigint", "int8":
		return "BIGINT"
	case "real", "float4", "float":
		return "REAL"
	case "double precision", "float8":
		return "DOUBLE PRECISION"
	case "numeric", "decimal":
		if col.NumPrecision != nil {
			if col.NumScale != nil {
				return fmt.Sprintf("NUMERIC(%d,%d)", *col.NumPrecision, *col.NumScale)
			}
			return fmt.Sprintf("NUMERIC(%d)", *col.NumPrecision)
		}
		return "NUMERIC"
	case "money":
		return "MONEY"
	case "boolean", "bool":
		return "BOOLEAN"
	case "character varying", "varchar":
		if col.CharMaxLen != nil {
			return fmt.Sprintf("VARCHAR(%d)", *col.CharMaxLen)
		}
		return "TEXT"
	case "character", "char":
		if col.CharMaxLen != nil {
			return fmt.Sprintf("CHAR(%d)", *col.CharMaxLen)
		}
		return "TEXT"
	case "text":
		return "TEXT"
	case "bytea":
		return "BYTEA"
	case "date":
		return "DATE"
	case "time with time zone", "timetz":
		return "TIMETZ"
	case "time without time zone", "time":
		return "TIME"
	case "timestamp with time zone", "timestamptz":
		return "TIMESTAMPTZ"
	case "timestamp without time zone", "timestamp":
		return "TIMESTAMP"
	case "interval":
		return "INTERVAL"
	case "uuid":
		return "UUID"
	case "json", "jsonb":
		return "JSONB"
	case "xml":
		return "XML"
	case "inet":
		return "INET"
	case "cidr":
		return "CIDR"
	case "macaddr":
		return "MACADDR"
	case "macaddr8":
		return "MACADDR8"
	case "bit":
		return "VARBIT"
	case "varbit", "bit varying":
		return "VARBIT"
	case "tsvector":
		return "TSVECTOR"
	case "tsquery":
		return "TSQUERY"
	case "point":
		return "POINT"
	case "line":
		return "LINE"
	case "lseg":
		return "LSEG"
	case "box":
		return "BOX"
	case "path":
		return "PATH"
	case "polygon":
		return "POLYGON"
	case "circle":
		return "CIRCLE"
	case "int4range":
		return "INT4RANGE"
	case "numrange":
		return "NUMRANGE"
	case "daterange":
		return "DATERANGE"
	case "tsrange":
		return "TSRANGE"
	case "tstzrange":
		return "TSTZRANGE"
	default:
		if baseType != "" {
			return strings.ToUpper(baseType)
		}
		return "TEXT"
	}
}
