package sqlgenerator

import (
	"analyticDataCenter/analytics-data-center/internal/domain/models"
	"fmt"
	"log/slog"
)

const (
	DbPostgres   = "postgres"
	DbClickhouse = "clickhouse"
)

// Универсальная функция выбора адаптера под базу
func GenerateQueryCreateTempTable(
	schema *models.View,
	logger *slog.Logger,
	dbName string,
) (models.Queries, []string, error) {
	switch dbName {
	case DbPostgres:
		return GenerateQueryCreateTempTablePostgres(schema, logger, dbName)
	case DbClickhouse:
		return GenerateQueryCreateTempTableClickhouse(schema, logger)
	default:
		return models.Queries{}, nil, fmt.Errorf("unsupported db: %s", dbName)
	}
}
