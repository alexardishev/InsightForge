package serviceanalytics

import (
	"context"
	"testing"

	"analyticDataCenter/analytics-data-center/internal/domain/models"
	"github.com/stretchr/testify/require"
)

func TestPrepareViewJoin(t *testing.T) {
	dwh := &mockDWH{columns: map[string][]string{"tmp1": {"id", "name"}}}
	svc := &AnalyticsDataCenterService{log: getTestLogger(), DWHProvider: dwh}
	res, err := svc.prepareViewJoin(context.Background(), []models.TempTable{{TempTableName: "tmp1"}}, "public")

	require.NoError(t, err)
	require.Len(t, res.TempTables, 1)
	require.Equal(t, "tmp1", res.TempTables[0].TempTableName)
	require.Equal(t, "id", res.TempTables[0].TempColumns[0].ColumnName)
}
