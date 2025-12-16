package sqlgenerator

import (
	"analyticDataCenter/analytics-data-center/internal/domain/models"
	"fmt"
	"strings"
)

func MapColumnTypeToPostgresDDL(col models.Column) string {
	dataType := strings.ToLower(strings.TrimSpace(col.DataType))
	udtName := strings.ToLower(strings.TrimSpace(col.UdtName))
	legacyType := strings.ToLower(strings.TrimSpace(col.Type))

	// 1) Определяем массивность по udt_name (_int4/_int8/_uuid/_text и т.д.)
	isArray := false
	if strings.HasPrefix(udtName, "_") {
		isArray = true
		udtName = strings.TrimPrefix(udtName, "_")
	} else if strings.HasPrefix(legacyType, "_") {
		// fallback только если udtName реально не пришел
		isArray = true
		legacyType = strings.TrimPrefix(legacyType, "_")
	}

	// 2) SAFE MODE: data_type = ARRAY, но без базового udt → TEXT[]
	// (в норме data_type может быть "ARRAY", а udt_name = "_int4". Это нормальный кейс)
	if dataType == "array" && udtName == "" && legacyType == "" {
		return "TEXT[]"
	}

	// 3) SAFE MODE: USER-DEFINED (citext/enum/domain/extension) => TEXT/TEXT[]
	// Только если data_type прямо user-defined
	if dataType == "user-defined" {
		if isArray {
			return "TEXT[]"
		}
		return "TEXT"
	}

	// 4) Выбираем базовый тип: сначала udtName (он точнее), потом dataType, потом legacy
	base := ""
	if udtName != "" && udtName != "array" {
		base = udtName
	} else if dataType != "" && dataType != "array" {
		base = dataType
	} else {
		base = legacyType
	}
	base = strings.ToLower(strings.TrimSpace(base))

	mapped := mapBasePostgresType(base, col)

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
		if col.NumPrecision != nil && col.NumScale != nil {
			return fmt.Sprintf("NUMERIC(%d,%d)", *col.NumPrecision, *col.NumScale)
		}
		if col.NumPrecision != nil {
			return fmt.Sprintf("NUMERIC(%d)", *col.NumPrecision)
		}
		return "NUMERIC"

	case "money":
		// SAFE DWH: MONEY лучше хранить как TEXT или NUMERIC.
		// Но раз у тебя уже так заведено - оставим MONEY как есть, или можешь TEXT.
		return "MONEY"

	case "boolean", "bool":
		return "BOOLEAN"

	case "character varying", "varchar":
		if col.CharMaxLen != nil {
			return fmt.Sprintf("VARCHAR(%d)", *col.CharMaxLen)
		}
		return "TEXT"

	case "character", "char", "bpchar":
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

	case "json":
		return "JSON"
	case "jsonb":
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
		// В schema из UI часто приходит просто "bit" без (n),
		// а вставка может быть B'11110000' и т.п.
		// BIT без длины = bit(1) => ломается.
		// Поэтому SAFE: если длина неизвестна — VARBIT.
		// Если длина пришла как bit(n) — она обрабатывается раньше (через разбор "(n)"), см. ниже.
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
		// Абсолютный safe fallback
		return "TEXT"
	}
}
