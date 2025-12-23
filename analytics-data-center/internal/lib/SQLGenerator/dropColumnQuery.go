package sqlgenerator

import (
	"fmt"
	"strings"

	"github.com/lib/pq"
)

func GenerateDropColumnQuery(dbName, schemaName, tableName, columnName string) (string, error) {
	switch strings.ToLower(dbName) {
	case "postgres", "postgresql":
		return fmt.Sprintf(
			"ALTER TABLE %s.%s DROP COLUMN IF EXISTS %s",
			pq.QuoteIdentifier(schemaName),
			pq.QuoteIdentifier(tableName),
			pq.QuoteIdentifier(columnName),
		), nil
	case "clickhouse":
		quote := func(identifier string) string {
			return fmt.Sprintf("`%s`", strings.ReplaceAll(identifier, "`", "``"))
		}

		return fmt.Sprintf(
			"ALTER TABLE %s.%s DROP COLUMN IF EXISTS %s",
			quote(schemaName),
			quote(tableName),
			quote(columnName),
		), nil
	default:
		return "", fmt.Errorf("unsupported db for drop column: %s", dbName)
	}
}
