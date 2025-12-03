package serviceanalytics

import (
	"context"
	"testing"

	"analyticDataCenter/analytics-data-center/internal/domain/models"

	"github.com/stretchr/testify/require"
)

func TestApplyColumnMismatchResolution_IgnoresOnly(t *testing.T) {
	schemaProvider := &mockSchemaProvider{mismatchGroups: map[int64]models.ColumnMismatchGroupWithItems{
		1: {
			Group: models.ColumnMismatchGroup{ID: 1, SchemaID: 1, DatabaseName: "db", SchemaName: "sch", TableName: "tbl", Status: models.ColumnMismatchStatusOpen},
		},
	}, views: map[int]models.View{1: sampleView()}}
	dwh := &mockDWH{}

	svc := &AnalyticsDataCenterService{
		log:                   getTestLogger(),
		SchemaProvider:        schemaProvider,
		ColumnMismatchStorage: schemaProvider,
		DWHProvider:           dwh,
		DWHDbName:             DbPostgres,
	}

	err := svc.ApplyColumnMismatchResolution(context.Background(), 1, models.ColumnMismatchResolution{
		Ignores: []string{"col1"},
	})

	require.NoError(t, err)
	require.Len(t, dwh.renameCalls, 0)
	require.Nil(t, schemaProvider.updated)
	require.Equal(t, models.ColumnMismatchStatusResolved, schemaProvider.mismatchGroups[1].Group.Status)
}

func TestApplyColumnMismatchResolution_Renames(t *testing.T) {
	view := sampleView()
	schemaProvider := &mockSchemaProvider{mismatchGroups: map[int64]models.ColumnMismatchGroupWithItems{
		1: {
			Group: models.ColumnMismatchGroup{ID: 1, SchemaID: 1, DatabaseName: "db", SchemaName: "sch", TableName: "tbl", Status: models.ColumnMismatchStatusOpen},
		},
	}, views: map[int]models.View{1: view}}
	dwh := &mockDWH{}

	svc := &AnalyticsDataCenterService{
		log:                   getTestLogger(),
		SchemaProvider:        schemaProvider,
		ColumnMismatchStorage: schemaProvider,
		DWHProvider:           dwh,
		DWHDbName:             DbPostgres,
	}

	err := svc.ApplyColumnMismatchResolution(context.Background(), 1, models.ColumnMismatchResolution{
		Renames: []models.RenameDecision{{OldName: "col1", NewName: "new_col"}},
	})

	require.NoError(t, err)
	require.Len(t, dwh.renameCalls, 1)
	updatedView := schemaProvider.updated[1]
	require.Equal(t, "new_col", updatedView.Sources[0].Schemas[0].Tables[0].Columns[0].Name)
	require.Equal(t, models.ColumnMismatchStatusResolved, schemaProvider.mismatchGroups[1].Group.Status)
}

func TestApplyColumnMismatchResolution_Deletes(t *testing.T) {
	view := sampleView()
	view.Sources[0].Schemas[0].Tables[0].Columns = append(view.Sources[0].Schemas[0].Tables[0].Columns, models.Column{Name: "legacy"})
	schemaProvider := &mockSchemaProvider{mismatchGroups: map[int64]models.ColumnMismatchGroupWithItems{
		1: {
			Group: models.ColumnMismatchGroup{ID: 1, SchemaID: 1, DatabaseName: "db", SchemaName: "sch", TableName: "tbl", Status: models.ColumnMismatchStatusOpen},
		},
	}, views: map[int]models.View{1: view}}
	dwh := &mockDWH{}

	svc := &AnalyticsDataCenterService{
		log:                   getTestLogger(),
		SchemaProvider:        schemaProvider,
		ColumnMismatchStorage: schemaProvider,
		DWHProvider:           dwh,
		DWHDbName:             DbPostgres,
	}

	err := svc.ApplyColumnMismatchResolution(context.Background(), 1, models.ColumnMismatchResolution{
		Deletes: []string{"legacy"},
	})

	require.NoError(t, err)
	updatedView := schemaProvider.updated[1]
	require.Len(t, updatedView.Sources[0].Schemas[0].Tables[0].Columns, 1)
	require.Equal(t, models.ColumnMismatchStatusResolved, schemaProvider.mismatchGroups[1].Group.Status)
}

func TestApplyColumnMismatchResolution_Mixed(t *testing.T) {
	view := sampleView()
	view.Sources[0].Schemas[0].Tables[0].Columns = append(view.Sources[0].Schemas[0].Tables[0].Columns, models.Column{Name: "legacy"})
	schemaProvider := &mockSchemaProvider{mismatchGroups: map[int64]models.ColumnMismatchGroupWithItems{
		1: {
			Group: models.ColumnMismatchGroup{ID: 1, SchemaID: 1, DatabaseName: "db", SchemaName: "sch", TableName: "tbl", Status: models.ColumnMismatchStatusOpen},
		},
	}, views: map[int]models.View{1: view}}
	dwh := &mockDWH{}

	svc := &AnalyticsDataCenterService{
		log:                   getTestLogger(),
		SchemaProvider:        schemaProvider,
		ColumnMismatchStorage: schemaProvider,
		DWHProvider:           dwh,
		DWHDbName:             DbPostgres,
	}

	err := svc.ApplyColumnMismatchResolution(context.Background(), 1, models.ColumnMismatchResolution{
		Renames: []models.RenameDecision{{OldName: "col1", NewName: "new_col"}},
		Deletes: []string{"legacy"},
		Ignores: []string{"col2"},
	})

	require.NoError(t, err)
	require.Len(t, dwh.renameCalls, 1)
	updatedView := schemaProvider.updated[1]
	require.Equal(t, "new_col", updatedView.Sources[0].Schemas[0].Tables[0].Columns[0].Name)
	require.Len(t, updatedView.Sources[0].Schemas[0].Tables[0].Columns, 1)
	require.Equal(t, models.ColumnMismatchStatusResolved, schemaProvider.mismatchGroups[1].Group.Status)
}

func sampleView() models.View {
	return models.View{
		Name: "tbl",
		Sources: []models.Source{
			{
				Name: "db",
				Schemas: []models.Schema{
					{
						Name: "sch",
						Tables: []models.Table{
							{
								Name:    "tbl",
								Columns: []models.Column{{Name: "col1"}},
							},
						},
					},
				},
			},
		},
	}
}
