package sqlgenerator

import (
	"analyticDataCenter/analytics-data-center/internal/domain/models"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
)

func TransformIndexDefToSQLExpression(indexExpression models.Index, schemaFrom, tableFrom, schemaTo, tableTo string, logger *slog.Logger) (string, error) {
	const op = "analytics.transformIndexDefToSQLExpression"
	logger = logger.With(slog.String("op", op))
	logger.Info("start operation")

	original := indexExpression.IndexDef
	originalIndexName := indexExpression.IndexName
	pattern := fmt.Sprintf(`ON\s+(ONLY\s+)?%s\.%s`, regexp.QuoteMeta(schemaFrom), regexp.QuoteMeta(tableFrom))
	re := regexp.MustCompile(pattern)

	replacement := fmt.Sprintf("ON %s.%s", schemaTo, tableTo)
	updated := re.ReplaceAllString(original, replacement)
	new := fmt.Sprintf("%s_%s", originalIndexName, tableTo)
	updated = strings.Replace(updated, originalIndexName, new, 1)
	if updated == original {
		logger.Warn("Регулярное выражение не сработало, замена не произведена",
			slog.String("pattern", pattern),
			slog.String("original", original),
		)
		return "", fmt.Errorf("не удалось заменить схему и таблицу в выражении: %s", original)
	}

	return updated, nil
}
