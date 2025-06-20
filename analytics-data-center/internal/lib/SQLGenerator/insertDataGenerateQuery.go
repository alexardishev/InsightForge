package sqlgenerator

import (
	"analyticDataCenter/analytics-data-center/internal/domain/models"
	"fmt"
	"log/slog"
)

func GenerateInsertDataQuery(
	view models.View,
	selectData []map[string]interface{},
	tempTableName string,
	logger *slog.Logger,
	dbtype string,
) (models.Query, error) {
	switch dbtype {
	case DbPostgres:
		return GenerateInsertDataQueryPostgres(view, selectData, tempTableName, logger)
	case DbClickhouse:
		return GenerateInsertDataQueryClickhouse(view, selectData, tempTableName, logger)
	default:
		return models.Query{}, fmt.Errorf("неизвестный тип базы: %s", dbtype)
	}
}
