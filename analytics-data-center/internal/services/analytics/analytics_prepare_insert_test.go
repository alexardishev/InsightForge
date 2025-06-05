package serviceanalytics

import (
	"context"
	"testing"

	"analyticDataCenter/analytics-data-center/internal/domain/models"
	"analyticDataCenter/analytics-data-center/internal/storage"

	"github.com/stretchr/testify/require"
)

func TestPrepareAndInsertData(t *testing.T) {
	oltp := &mockOLTP{selectResult: []map[string]interface{}{{"id": 1, "name": "A"}}}
	factory := &mockFactory{store: map[string]storage.OLTPDB{"db1": oltp}}
	dwh := &mockDWH{columns: map[string][]string{"tmp_users": {"id", "name"}}}
	view := &models.View{
		Name: "v",
		Sources: []models.Source{{
			Name: "db1",
			Schemas: []models.Schema{{
				Name: "public",
				Tables: []models.Table{{
					Name: "users",
					Columns: []models.Column{
						{Name: "id", IsPrimaryKey: true},
						{Name: "name"},
					},
				}},
			}},
		}},
	}
	svc := &AnalyticsDataCenterService{log: getTestLogger(), OLTPFactory: factory, DWHProvider: dwh}
	data := []models.CountInsertData{{TableName: "users", Count: 1, DataBaseName: "db1", TempTableName: "tmp_users"}}

	ok, err := svc.prepareAndInsertData(context.Background(), &data, view)

	require.True(t, ok)
	require.NoError(t, err)
	require.NotEmpty(t, dwh.insertCalls)
	require.Equal(t, 1, dwh.mergeCalls)
	require.Equal(t, []string{"tmp_users"}, dwh.deleteCalls)
}
