package serviceanalytics

import (
	"context"
	"fmt"
	"testing"

	"analyticDataCenter/analytics-data-center/internal/domain/models"

	"github.com/stretchr/testify/require"
)

func TestCreateTempTablesSuccess(t *testing.T) {
	dwh := &mockDWH{}
	svc := &AnalyticsDataCenterService{log: getTestLogger(), DWHProvider: dwh}
	queries := models.Queries{Queries: []models.Query{{TableName: "t1"}, {TableName: "t2"}}}

	err := svc.createTempTables(context.Background(), queries)

	require.NoError(t, err)
	require.Equal(t, []string{"t1", "t2"}, dwh.createCalls)
	require.Empty(t, dwh.deleteCalls)
}

func TestCreateTempTablesErrorCleanup(t *testing.T) {
	dwh := &mockDWH{createErrors: map[string]error{"t2": fmt.Errorf("fail")}}
	svc := &AnalyticsDataCenterService{log: getTestLogger(), DWHProvider: dwh}
	queries := models.Queries{Queries: []models.Query{{TableName: "t1"}, {TableName: "t2"}}}

	err := svc.createTempTables(context.Background(), queries)

	require.Error(t, err)
	require.Equal(t, []string{"t1", "t2"}, dwh.createCalls)
	require.Equal(t, []string{"t1", "t2"}, dwh.deleteCalls)
}

func TestDeleteTempTables(t *testing.T) {
	dwh := &mockDWH{}
	svc := &AnalyticsDataCenterService{log: getTestLogger(), DWHProvider: dwh}

	err := svc.DeleteTempTables(context.Background(), []string{"t1", "t2"})

	require.NoError(t, err)
	require.Equal(t, []string{"t1", "t2"}, dwh.deleteCalls)
}

func TestDeleteTempTablesError(t *testing.T) {
	dwh := &mockDWH{deleteErrors: map[string]error{"t2": fmt.Errorf("fail")}}
	svc := &AnalyticsDataCenterService{log: getTestLogger(), DWHProvider: dwh}

	err := svc.DeleteTempTables(context.Background(), []string{"t1", "t2"})

	require.Error(t, err)
	require.Equal(t, []string{"t1"}, dwh.deleteCalls[:1])
}
