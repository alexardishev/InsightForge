package sqlgenerator

import (
	"analyticDataCenter/analytics-data-center/internal/domain/models"
	"log/slog"
)

func CreateViewQuery(schema models.View, tempTablesNames []string, tempColumnsNames []string, logger *slog.Logger) error {
	const op = "sqlgenerator.GenerateQueryCreateViewQuery"
	logger = logger.With(slog.String("op", op))
	logger.Info("start operation")
	// var b strings.Builder
	// viewName := schema.Name

	// b.WriteString()

	return nil
}
