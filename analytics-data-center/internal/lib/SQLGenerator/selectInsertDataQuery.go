package sqlgenerator

import (
	"analyticDataCenter/analytics-data-center/internal/domain/models"
	"fmt"
	"log/slog"
)

const (
	transformTypeJSON           = "JSON"
	transformTypeFieldTransform = "FieldTransform"
)

func GenerateSelectInsertDataQuery(
	view models.View,
	start int64,
	end int64,
	tableName string,
	logger *slog.Logger,
	dbtype string,
) (models.Query, error) {
	switch dbtype {
	case DbPostgres:
		return GenerateSelectInsertDataQueryPostgres(view, start, end, tableName, logger)
	case DbClickhouse:
		return GenerateSelectInsertDataQueryClickhouse(view, start, end, tableName, logger)
	default:
		return models.Query{}, fmt.Errorf("неизвестный тип базы: %s", dbtype)
	}
}
