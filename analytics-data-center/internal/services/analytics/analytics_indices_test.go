package serviceanalytics

import (
	"context"
	"testing"

	"analyticDataCenter/analytics-data-center/internal/domain/models"
	"analyticDataCenter/analytics-data-center/internal/storage"

	"github.com/stretchr/testify/require"
)

func TestTransferIndices(t *testing.T) {
	indexSQL := "CREATE INDEX idx ON public.t1(id)"
	oltp := &mockOLTP{indexResult: models.Indexes{Indexes: []models.Index{{IndexName: "idx", IndexDef: indexSQL}}}}
	factory := &mockFactory{store: map[string]storage.OLTPDB{"db1": oltp}}
	dwh := &mockDWH{}
	view := &models.View{Sources: []models.Source{{Name: "db1", Schemas: []models.Schema{{Name: "public", Tables: []models.Table{{Name: "t1"}}}}}}}
	svc := &AnalyticsDataCenterService{log: getTestLogger(), OLTPFactory: factory, DWHProvider: dwh}

	err := svc.transferIndixesAndConstraint(context.Background(), view, "postgres")

	require.NoError(t, err)
	require.NotEmpty(t, dwh.indexCalls)
}
