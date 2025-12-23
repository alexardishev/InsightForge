package sqlgenerator

import (
	"analyticDataCenter/analytics-data-center/internal/domain/models"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"
)

func TransformIndexDefToSQLExpression(indexExpression models.Index, schemaFrom string, tableFrom string, schemaTo string, tableTo string, logger *slog.Logger) (string, error) {
	const op = "analytics.transformIndexDefToSQLExpression"
	logger = logger.With(slog.String("op", op))
	logger.Info("start operation")

	original := indexExpression.IndexDef
	originalIndexName := indexExpression.IndexName
	pattern := fmt.Sprintf(`ON\s+(ONLY\s+)?%s\.%s`, regexp.QuoteMeta(schemaFrom), regexp.QuoteMeta(tableFrom))
	re := regexp.MustCompile(pattern)

	replacement := fmt.Sprintf("ON %s.%s", schemaTo, tableTo)
	updated := re.ReplaceAllString(original, replacement)
	timePrefix := time.Now().UTC().Format("20060102_150405")
	safeIndexName := sanitizeIndexName(originalIndexName)
	new := fmt.Sprintf("idx_%s_%s", timePrefix, safeIndexName)
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

func sanitizeIndexName(name string) string {
	re := regexp.MustCompile(`[^\w]+`)
	sanitized := re.ReplaceAllString(name, "_")
	return strings.Trim(sanitized, "_")
}
