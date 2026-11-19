package sqlgenerator

import (
	"analyticDataCenter/analytics-data-center/internal/domain/models"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
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
	taskID uuid.UUID,
) (models.Queries, []string, error) {
	switch dbName {
	case DbPostgres:
		return GenerateQueryCreateTempTablePostgres(schema, logger, taskID)
	case DbClickhouse:
		return GenerateQueryCreateTempTableClickhouse(schema, logger, taskID)
	default:
		return models.Queries{}, nil, fmt.Errorf("unsupported db: %s", dbName)
	}
}

func tempTableName(taskID uuid.UUID, tableIndex int) string {
	return fmt.Sprintf("temp_%x_%d", taskID[:], tableIndex)
}
