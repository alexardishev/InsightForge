package sqlgenerator

import (
	"fmt"
	"strings"

	"github.com/lib/pq"
)

func GenerateRenameColumnQuery(dbName, schemaName, tableName, oldName, newName string) (string, error) {
	switch strings.ToLower(dbName) {
	case "postgres", "postgresql":
		return fmt.Sprintf(
			"ALTER TABLE %s.%s RENAME COLUMN %s TO %s",
			pq.QuoteIdentifier(schemaName),
			pq.QuoteIdentifier(tableName),
			pq.QuoteIdentifier(oldName),
			pq.QuoteIdentifier(newName),
		), nil
	case "clickhouse":
		quote := func(identifier string) string {
			return fmt.Sprintf("`%s`", strings.ReplaceAll(identifier, "`", "``"))
		}

		return fmt.Sprintf(
			"ALTER TABLE %s.%s RENAME COLUMN %s TO %s",
			quote(schemaName),
			quote(tableName),
			quote(oldName),
			quote(newName),
		), nil
	default:
		return "", fmt.Errorf("unsupported db for rename column: %s", dbName)
	}
}
