package serviceanalytics

import (
	"context"
	"testing"

	"analyticDataCenter/analytics-data-center/internal/domain/models"
	smtpsender "analyticDataCenter/analytics-data-center/internal/services/smtrsender"
	"analyticDataCenter/analytics-data-center/internal/storage"

	"github.com/stretchr/testify/require"
)

type mockSchemaProvider struct {
	views     map[int]models.View
	schems    []int
	updated   map[int]models.View
	updateErr error
}

func (m *mockSchemaProvider) CreateTask(context.Context, string, string) error { return nil }
func (m *mockSchemaProvider) GetTask(context.Context, string) (models.Task, error) {
	return models.Task{}, nil
}
func (m *mockSchemaProvider) ChangeStatusTask(context.Context, string, string, string) error {
	return nil
}
func (m *mockSchemaProvider) GetTasks(context.Context, models.TaskFilter) ([]models.Task, error) {
	return nil, nil
}

func (m *mockSchemaProvider) GetView(_ context.Context, idView int64) (models.View, error) {
	if v, ok := m.views[int(idView)]; ok {
		return v, nil
	}
	return models.View{}, storage.ErrSchemaNotFound
}

func (m *mockSchemaProvider) GetSchems(context.Context, string, string, string) ([]int, error) {
	return m.schems, nil
}

func (m *mockSchemaProvider) UpdateView(_ context.Context, view models.View, schemaId int) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	if m.updated == nil {
		m.updated = make(map[int]models.View)
	}
	m.updated[schemaId] = view
	return nil
}

func (m *mockSchemaProvider) UploadView(context.Context, models.View) (int64, error) {
	return 0, nil
}

func (m *mockSchemaProvider) ListTopics(context.Context) ([]string, error) { return nil, nil }

func TestCheckColumnInTables_RenameFromOLTP(t *testing.T) {
	ctx := context.Background()

	view := models.View{
		Name: "users_view",
		Sources: []models.Source{{
			Name: "db1",
			Schemas: []models.Schema{{
				Name: "public",
				Tables: []models.Table{{
					Name:    "users",
					Columns: []models.Column{{Name: "id"}, {Name: "name"}, {Name: "plan_id"}},
				}},
			}},
		}},
	}

	schemaProvider := &mockSchemaProvider{views: map[int]models.View{1: view}}
	dwh := &mockDWH{columns: map[string][]string{"users_view": {"id", "name", "plan_id"}}}
	oltp := &mockOLTP{columns: []models.Column{{Name: "id", Type: "integer"}, {Name: "title", Type: "text"}, {Name: "plan_id", Type: "integer"}}}
	factory := &mockFactory{store: map[string]storage.OLTPDB{"db1": oltp}}
	smtp := smtpsender.SMTP{EventQueueSMTP: make(chan models.Event, 1)}

	svc := &AnalyticsDataCenterService{
		log:            getTestLogger(),
		SchemaProvider: schemaProvider,
		DWHProvider:    dwh,
		OLTPFactory:    factory,
		DWHDbName:      DbPostgres,
		SMTPClient:     smtp,
	}

	err := svc.checkColumnInTables(ctx, nil, map[string]interface{}{"title": "new"}, "db1", "public", "users", []int{1})

	require.NoError(t, err)
	require.Len(t, dwh.renameCalls, 1)
	updated := schemaProvider.updated[1]
	require.Equal(t, []string{"id", "title", "plan_id"}, columnNames(updated))
	require.False(t, updated.Sources[0].Schemas[0].Tables[0].Columns[1].IsDeleted)
	require.Len(t, smtp.EventQueueSMTP, 0)
}

func TestCheckColumnInTables_SetDeletedWhenMissingInOLTP(t *testing.T) {
	ctx := context.Background()

	view := models.View{
		Name: "users_view",
		Sources: []models.Source{{
			Name: "db1",
			Schemas: []models.Schema{{
				Name: "public",
				Tables: []models.Table{{
					Name:    "users",
					Columns: []models.Column{{Name: "id"}, {Name: "name"}, {Name: "plan_id"}},
				}},
			}},
		}},
	}

	schemaProvider := &mockSchemaProvider{views: map[int]models.View{1: view}}
	dwh := &mockDWH{columns: map[string][]string{"users_view": {"id", "name", "plan_id"}}}
	oltp := &mockOLTP{columns: []models.Column{{Name: "id", Type: "integer"}, {Name: "plan_id", Type: "integer"}}}
	factory := &mockFactory{store: map[string]storage.OLTPDB{"db1": oltp}}
	smtp := smtpsender.SMTP{EventQueueSMTP: make(chan models.Event, 1)}

	svc := &AnalyticsDataCenterService{
		log:            getTestLogger(),
		SchemaProvider: schemaProvider,
		DWHProvider:    dwh,
		OLTPFactory:    factory,
		DWHDbName:      DbPostgres,
		SMTPClient:     smtp,
	}

	err := svc.checkColumnInTables(ctx, nil, map[string]interface{}{"plan_id": 1}, "db1", "public", "users", []int{1})

	require.NoError(t, err)
	require.Len(t, dwh.renameCalls, 0)
	updated := schemaProvider.updated[1]
	require.True(t, updated.Sources[0].Schemas[0].Tables[0].Columns[1].IsDeleted)
	require.Len(t, smtp.EventQueueSMTP, 1)
}

func columnNames(view models.View) []string {
	columns := view.Sources[0].Schemas[0].Tables[0].Columns
	var names []string
	for _, c := range columns {
		names = append(names, c.Name)
	}
	return names
}
