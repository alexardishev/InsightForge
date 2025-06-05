package serviceanalytics

import (
	"context"
	"fmt"
	"testing"

	"analyticDataCenter/analytics-data-center/internal/domain/models"
	"analyticDataCenter/analytics-data-center/internal/storage"

	"github.com/stretchr/testify/require"
)

func TestGetCountInsertData(t *testing.T) {
	oltp := &mockOLTP{countResult: 5}
	factory := &mockFactory{store: map[string]storage.OLTPDB{"db1": oltp}}
	svc := &AnalyticsDataCenterService{log: getTestLogger(), OLTPFactory: factory}

	view := models.View{Sources: []models.Source{{Name: "db1", Schemas: []models.Schema{{Tables: []models.Table{{Name: "users"}}}}}}}
	res, err := svc.getCountInsertData(context.Background(), view, []string{"tmp"})

	require.NoError(t, err)
	require.Len(t, res, 1)
	require.Equal(t, int64(5), res[0].Count)
	require.Equal(t, "tmp", res[0].TempTableName)
}

func TestGetCountInsertDataStorageError(t *testing.T) {
	factory := &mockFactory{err: fmt.Errorf("no storage")}
	svc := &AnalyticsDataCenterService{log: getTestLogger(), OLTPFactory: factory}
	view := models.View{Sources: []models.Source{{Name: "db1", Schemas: []models.Schema{{Tables: []models.Table{{Name: "users"}}}}}}}

	_, err := svc.getCountInsertData(context.Background(), view, []string{"tmp"})

	require.Error(t, err)
}
