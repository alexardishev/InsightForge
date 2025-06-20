package sqlgenerator

import (
	"analyticDataCenter/analytics-data-center/internal/domain/models"
	"fmt"
	"log/slog"
	"strings"
	"unicode/utf8"
)

func CreateViewQuery(
	schema models.View,
	viewJoin models.ViewJoinTable,
	logger *slog.Logger,
	dbType string,
) (models.Query, error) {
	switch dbType {
	case DbPostgres:
		return CreateViewQueryPostgres(schema, viewJoin, logger)
	case DbClickhouse:
		return CreateViewQueryClickhouse(schema, viewJoin, logger)
	default:
		return models.Query{}, fmt.Errorf("неподдерживаемый тип DWH: %s", dbType)
	}
}

func CleanAndTrim(input string, maxLen int) string {
	cleaned := strings.TrimSpace(input)
	cleaned = strings.Join(strings.Fields(cleaned), "_")
	if utf8.RuneCountInString(cleaned) > maxLen {
		runes := []rune(cleaned)
		return string(runes[len(runes)-maxLen:])
	}
	return cleaned
}
