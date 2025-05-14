package sqlgenerator

import (
	"analyticDataCenter/analytics-data-center/internal/domain/models"
	"fmt"
	"log/slog"
)

func TransformConstraintToExpression(constraint models.Constarint, schemaFrom string, tableFrom string, logger *slog.Logger) string {
	const op = "analytics.TransformConstraintToExpression"
	logger = logger.With(slog.String("op", op))
	logger.Info("start operation")

	constraintName := fmt.Sprintf(`"%s_%s"`, tableFrom, constraint.ConstarintName)

	// Экранируем имена схемы и таблицы
	safeSchema := fmt.Sprintf(`"%s"`, schemaFrom)
	safeTable := fmt.Sprintf(`"%s"`, tableFrom)

	query := fmt.Sprintf(
		`ALTER TABLE %s.%s ADD CONSTRAINT %s %s;`,
		safeSchema,
		safeTable,
		constraintName,
		constraint.Defenition,
	)
	return query
}
