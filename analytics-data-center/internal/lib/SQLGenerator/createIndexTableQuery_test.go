package sqlgenerator

import (
	"analyticDataCenter/analytics-data-center/internal/domain/models"
	"log/slog"
	"os"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTransformIndexDefToSQLExpression_RenamesIndexWithTimestamp(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	index := models.Index{
		IndexName: "my complex index",
		IndexDef:  "CREATE INDEX my complex index ON public.users USING btree (id)",
	}

	query, err := TransformIndexDefToSQLExpression(index, "public", "users", "public", "analytics_view", logger)

	require.NoError(t, err)
	require.Contains(t, query, "ON public.analytics_view")

	pattern := regexp.MustCompile(`CREATE INDEX \d{8}_\d{6}_my_complex_index ON`)
	require.True(t, pattern.MatchString(query), "index name should be prefixed with timestamp and sanitized")
}
