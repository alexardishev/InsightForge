package serviceanalytics

import (
	"context"
	"fmt"
	"testing"

	"analyticDataCenter/analytics-data-center/internal/domain/models"
	smtpsender "analyticDataCenter/analytics-data-center/internal/services/smtrsender"
	"analyticDataCenter/analytics-data-center/internal/storage"

	"github.com/stretchr/testify/require"
)

type mockSchemaProvider struct {
	views            map[int]models.View
	schems           []int
	updated          map[int]models.View
	updateErr        error
	suggestions      []models.ColumnRenameSuggestion
	hasSuggestions   map[string]bool
	hasSuggestionErr error
	mismatchGroups   map[int64]models.ColumnMismatchGroupWithItems
	mismatchSeq      int64
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

func (m *mockSchemaProvider) ListViews(context.Context) ([]models.SchemaInfo, error) {
	return nil, nil
}

func (m *mockSchemaProvider) ListTopics(context.Context) ([]string, error) { return nil, nil }

func (m *mockSchemaProvider) CreateSuggestion(_ context.Context, s models.ColumnRenameSuggestion) error {
	m.suggestions = append(m.suggestions, s)
	return nil
}

func (m *mockSchemaProvider) ListSuggestions(context.Context, models.ColumnRenameSuggestionFilter) ([]models.ColumnRenameSuggestion, error) {
	return m.suggestions, nil
}

func (m *mockSchemaProvider) GetSuggestionByID(_ context.Context, id int64) (models.ColumnRenameSuggestion, error) {
	for _, suggestion := range m.suggestions {
		if suggestion.ID == id {
			return suggestion, nil
		}
	}

	return models.ColumnRenameSuggestion{}, storage.ErrSuggestionNotFound
}

func (m *mockSchemaProvider) DeleteSuggestionByID(_ context.Context, id int64) error {
	for idx, suggestion := range m.suggestions {
		if suggestion.ID == id {
			m.suggestions = append(m.suggestions[:idx], m.suggestions[idx+1:]...)
			return nil
		}
	}

	return storage.ErrSuggestionNotFound
}

func (m *mockSchemaProvider) HasSuggestion(_ context.Context, schemaID int64, database, schema, table string) (bool, error) {
	if m.hasSuggestionErr != nil {
		return false, m.hasSuggestionErr
	}

	key := fmt.Sprintf("%d:%s:%s:%s", schemaID, database, schema, table)
	if m.hasSuggestions != nil {
		return m.hasSuggestions[key], nil
	}

	for _, s := range m.suggestions {
		if s.SchemaID == schemaID && s.DatabaseName == database && s.SchemaName == schema && s.TableName == table {
			return true, nil
		}
	}

	return false, nil
}

func (m *mockSchemaProvider) CreateMismatchGroup(_ context.Context, group models.ColumnMismatchGroup, items []models.ColumnMismatchItem) (int64, error) {
	if m.mismatchGroups == nil {
		m.mismatchGroups = make(map[int64]models.ColumnMismatchGroupWithItems)
	}
	m.mismatchSeq++
	group.ID = m.mismatchSeq
	m.mismatchGroups[group.ID] = models.ColumnMismatchGroupWithItems{Group: group, Items: items}
	return group.ID, nil
}

func (m *mockSchemaProvider) ReplaceMismatchItems(_ context.Context, groupID int64, items []models.ColumnMismatchItem) error {
	g, ok := m.mismatchGroups[groupID]
	if !ok {
		return storage.ErrMismatchNotFound
	}
	g.Items = items
	m.mismatchGroups[groupID] = g
	return nil
}

func (m *mockSchemaProvider) GetOpenMismatchGroup(_ context.Context, schemaID int64, database, schema, table string) (models.ColumnMismatchGroupWithItems, error) {
	for _, g := range m.mismatchGroups {
		if g.Group.SchemaID == schemaID && g.Group.DatabaseName == database && g.Group.SchemaName == schema && g.Group.TableName == table && g.Group.Status == models.ColumnMismatchStatusOpen {
			return g, nil
		}
	}
	return models.ColumnMismatchGroupWithItems{}, storage.ErrMismatchNotFound
}

func (m *mockSchemaProvider) ListMismatchGroups(context.Context, models.ColumnMismatchFilter) ([]models.ColumnMismatchGroup, error) {
	var groups []models.ColumnMismatchGroup
	for _, g := range m.mismatchGroups {
		groups = append(groups, g.Group)
	}
	return groups, nil
}

func (m *mockSchemaProvider) GetMismatchGroup(_ context.Context, id int64) (models.ColumnMismatchGroupWithItems, error) {
	g, ok := m.mismatchGroups[id]
	if !ok {
		return models.ColumnMismatchGroupWithItems{}, storage.ErrMismatchNotFound
	}
	return g, nil
}

func (m *mockSchemaProvider) ResolveMismatchGroup(_ context.Context, id int64) error {
	g, ok := m.mismatchGroups[id]
	if !ok {
		return storage.ErrMismatchNotFound
	}
	g.Group.Status = models.ColumnMismatchStatusResolved
	m.mismatchGroups[id] = g
	return nil
}

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
		log:                     getTestLogger(),
		SchemaProvider:          schemaProvider,
		RenameSuggestionStorage: schemaProvider,
		ColumnMismatchStorage:   schemaProvider,
		DWHProvider:             dwh,
		OLTPFactory:             factory,
		DWHDbName:               DbPostgres,
		SMTPClient:              smtp,
	}

	err := svc.checkColumnInTables(ctx, nil, map[string]interface{}{"title": "new"}, "db1", "public", "users", []int{1})

	require.NoError(t, err)
	require.Len(t, schemaProvider.mismatchGroups, 1)
	for _, group := range schemaProvider.mismatchGroups {
		require.Equal(t, models.ColumnMismatchStatusOpen, group.Group.Status)
		require.NotEmpty(t, group.Items)

		foundSchemaOnly := false
		for _, item := range group.Items {
			if item.Type == models.ColumnMismatchTypeSchemaOnly {
				foundSchemaOnly = true
			}
		}

		require.True(t, foundSchemaOnly, "schema_only item expected")
	}
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
		log:                     getTestLogger(),
		SchemaProvider:          schemaProvider,
		RenameSuggestionStorage: schemaProvider,
		ColumnMismatchStorage:   schemaProvider,
		DWHProvider:             dwh,
		OLTPFactory:             factory,
		DWHDbName:               DbPostgres,
		SMTPClient:              smtp,
	}

	err := svc.checkColumnInTables(ctx, nil, map[string]interface{}{"plan_id": 1}, "db1", "public", "users", []int{1})

	require.NoError(t, err)
	require.Len(t, schemaProvider.mismatchGroups, 1)
	for _, group := range schemaProvider.mismatchGroups {
		require.Equal(t, models.ColumnMismatchStatusOpen, group.Group.Status)
		require.Len(t, group.Items, 1)
		require.Equal(t, models.ColumnMismatchTypeSchemaOnly, group.Items[0].Type)
	}
}

func TestCheckColumnInTables_IgnoresOtherTablesColumns(t *testing.T) {
	ctx := context.Background()

	view := models.View{
		Name: "combined_view",
		Sources: []models.Source{{
			Name: "db1",
			Schemas: []models.Schema{{
				Name: "public",
				Tables: []models.Table{
					{
						Name:    "users",
						Columns: []models.Column{{Name: "id"}, {Name: "name"}},
					},
					{
						Name:    "orders",
						Columns: []models.Column{{Name: "order_id"}, {Name: "user_id"}, {Name: "total"}},
					},
				},
			}},
		}},
	}

	schemaProvider := &mockSchemaProvider{views: map[int]models.View{1: view}}
	dwh := &mockDWH{columns: map[string][]string{"combined_view": {"id", "name", "order_id", "user_id", "total"}}}
	oltp := &mockOLTP{columns: []models.Column{{Name: "id", Type: "integer"}, {Name: "name", Type: "text"}}}
	factory := &mockFactory{store: map[string]storage.OLTPDB{"db1": oltp}}
	smtp := smtpsender.SMTP{EventQueueSMTP: make(chan models.Event, 1)}

	svc := &AnalyticsDataCenterService{
		log:                     getTestLogger(),
		SchemaProvider:          schemaProvider,
		RenameSuggestionStorage: schemaProvider,
		ColumnMismatchStorage:   schemaProvider,
		DWHProvider:             dwh,
		OLTPFactory:             factory,
		DWHDbName:               DbPostgres,
		SMTPClient:              smtp,
	}

	err := svc.checkColumnInTables(ctx, nil, map[string]interface{}{"id": 1}, "db1", "public", "users", []int{1})

	require.NoError(t, err)
	require.Len(t, schemaProvider.mismatchGroups, 0)
}

func columnNames(view models.View) []string {
	columns := view.Sources[0].Schemas[0].Tables[0].Columns
	var names []string
	for _, c := range columns {
		names = append(names, c.Name)
	}
	return names
}
