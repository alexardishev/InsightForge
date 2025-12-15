package dbhandlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
	"unsafe"

	"analyticDataCenter/analytics-data-center/internal/domain/models"
	loggerpkg "analyticDataCenter/analytics-data-center/internal/logger"
	serviceanalytics "analyticDataCenter/analytics-data-center/internal/services/analytics"
	"analyticDataCenter/analytics-data-center/internal/storage"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

type testSchemaProvider struct {
	views          map[int]models.View
	updated        map[int]models.View
	mismatchGroups map[int64]models.ColumnMismatchGroupWithItems
	lastFilter     models.ColumnMismatchFilter
}

func (m *testSchemaProvider) CreateTask(context.Context, string, string) error { return nil }
func (m *testSchemaProvider) GetTask(context.Context, string) (models.Task, error) {
	return models.Task{}, nil
}
func (m *testSchemaProvider) ChangeStatusTask(context.Context, string, string, string) error {
	return nil
}
func (m *testSchemaProvider) GetTasks(context.Context, models.TaskFilter) ([]models.Task, error) {
	return nil, nil
}

func (m *testSchemaProvider) GetView(_ context.Context, idView int64) (models.View, error) {
	if v, ok := m.views[int(idView)]; ok {
		return v, nil
	}
	return models.View{}, storage.ErrSchemaNotFound
}
func (m *testSchemaProvider) GetSchems(context.Context, string, string, string) ([]int, error) {
	return nil, nil
}
func (m *testSchemaProvider) UpdateView(_ context.Context, view models.View, schemaId int) error {
	if m.updated == nil {
		m.updated = make(map[int]models.View)
	}
	m.updated[schemaId] = view
	return nil
}
func (m *testSchemaProvider) UploadView(context.Context, models.View) (int64, error) { return 0, nil }
func (m *testSchemaProvider) ListViews(context.Context) ([]models.SchemaInfo, error) { return nil, nil }
func (m *testSchemaProvider) ListTopics(context.Context) ([]string, error)           { return nil, nil }

func (m *testSchemaProvider) CreateSuggestion(context.Context, models.ColumnRenameSuggestion) error {
	return nil
}
func (m *testSchemaProvider) ListSuggestions(context.Context, models.ColumnRenameSuggestionFilter) ([]models.ColumnRenameSuggestion, error) {
	return nil, nil
}
func (m *testSchemaProvider) HasSuggestion(context.Context, int64, string, string, string) (bool, error) {
	return false, nil
}
func (m *testSchemaProvider) GetSuggestionByID(context.Context, int64) (models.ColumnRenameSuggestion, error) {
	return models.ColumnRenameSuggestion{}, storage.ErrSuggestionNotFound
}
func (m *testSchemaProvider) DeleteSuggestionByID(context.Context, int64) error {
	return storage.ErrSuggestionNotFound
}

func (m *testSchemaProvider) CreateMismatchGroup(context.Context, models.ColumnMismatchGroup, []models.ColumnMismatchItem) (int64, error) {
	return 0, nil
}
func (m *testSchemaProvider) ReplaceMismatchItems(context.Context, int64, []models.ColumnMismatchItem) error {
	return nil
}
func (m *testSchemaProvider) GetOpenMismatchGroup(context.Context, int64, string, string, string) (models.ColumnMismatchGroupWithItems, error) {
	return models.ColumnMismatchGroupWithItems{}, storage.ErrMismatchNotFound
}
func (m *testSchemaProvider) ListMismatchGroups(_ context.Context, filter models.ColumnMismatchFilter) ([]models.ColumnMismatchGroup, error) {
	m.lastFilter = filter
	return m.collectGroups(), nil
}
func (m *testSchemaProvider) GetMismatchGroup(_ context.Context, id int64) (models.ColumnMismatchGroupWithItems, error) {
	if g, ok := m.mismatchGroups[id]; ok {
		return g, nil
	}
	return models.ColumnMismatchGroupWithItems{}, storage.ErrMismatchNotFound
}
func (m *testSchemaProvider) ResolveMismatchGroup(_ context.Context, id int64) error {
	if g, ok := m.mismatchGroups[id]; ok {
		now := time.Now()
		g.Group.Status = models.ColumnMismatchStatusResolved
		g.Group.ResolvedAt = &now
		m.mismatchGroups[id] = g
		return nil
	}
	return storage.ErrMismatchNotFound
}

func (m *testSchemaProvider) collectGroups() []models.ColumnMismatchGroup {
	var groups []models.ColumnMismatchGroup
	for _, g := range m.mismatchGroups {
		groups = append(groups, g.Group)
	}
	return groups
}

func (m *testSchemaProvider) setFilter(filter models.ColumnMismatchFilter) {
	m.lastFilter = filter
}

// DWH mock
type testDWH struct{ renameCalls []string }

func (d *testDWH) CreateTempTable(context.Context, string, string) error { return nil }
func (d *testDWH) DeleteTempTable(context.Context, string) error         { return nil }
func (d *testDWH) CreateIndex(context.Context, string) error             { return nil }
func (d *testDWH) CreateConstraint(context.Context, string) error        { return nil }
func (d *testDWH) RenameColumn(_ context.Context, query string) error {
	d.renameCalls = append(d.renameCalls, query)
	return nil
}
func (d *testDWH) InsertDataToDWH(context.Context, string) error { return nil }
func (d *testDWH) GetColumnsTables(context.Context, string, string) ([]string, error) {
	return nil, nil
}
func (d *testDWH) MergeTempTables(context.Context, string) error     { return nil }
func (d *testDWH) ReplicaIdentityFull(context.Context, string) error { return nil }
func (d *testDWH) InsertOrUpdateTransactional(context.Context, string, map[string]interface{}, []string) error {
	return nil
}

func setupHandlerForTests() (*DBHandlers, *testSchemaProvider, *testDWH) {
	schemaProvider := &testSchemaProvider{
		views: map[int]models.View{1: {
			Name: "tbl",
			Sources: []models.Source{{
				Name: "db",
				Schemas: []models.Schema{{
					Name: "sch",
					Tables: []models.Table{{
						Name:    "tbl",
						Columns: []models.Column{{Name: "col1"}},
					}},
				}},
			}},
		}},
		mismatchGroups: map[int64]models.ColumnMismatchGroupWithItems{
			1: {
				Group: models.ColumnMismatchGroup{ID: 1, SchemaID: 1, DatabaseName: "db", SchemaName: "sch", TableName: "tbl", Status: models.ColumnMismatchStatusOpen},
				Items: []models.ColumnMismatchItem{{ID: 1, GroupID: 1, Type: models.ColumnMismatchTypeSchemaOnly, OldColumnName: ptr("col1")}},
			},
		},
	}

	dwh := &testDWH{}
	svc := &serviceanalytics.AnalyticsDataCenterService{
		SchemaProvider:        schemaProvider,
		ColumnMismatchStorage: schemaProvider,
		DWHProvider:           dwh,
		DWHDbName:             serviceanalytics.DbPostgres,
	}
	logField := reflect.ValueOf(svc).Elem().FieldByName("log")
	reflect.NewAt(logField.Type(), unsafe.Pointer(logField.UnsafeAddr())).Elem().Set(reflect.ValueOf(loggerpkg.New("test", "ru")))

	handler := NewDBHandler(loggerpkg.New("test", "ru"), svc)
	return handler, schemaProvider, dwh
}

func TestGetColumnMismatchGroups(t *testing.T) {
	handler, schemaProvider, _ := setupHandlerForTests()

	r := chi.NewRouter()
	r.Get("/api/column-mismatch-groups", handler.GetColumnMismatchGroups)

	req := httptest.NewRequest(http.MethodGet, "/api/column-mismatch-groups?status=open&database=db&schema=sch&table=tbl&limit=10&offset=5", nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var resp struct {
		Items  []models.ColumnMismatchGroup `json:"items"`
		Limit  int                          `json:"limit"`
		Offset int                          `json:"offset"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	require.Len(t, resp.Items, 1)
	require.Equal(t, 10, resp.Limit)
	require.Equal(t, 5, resp.Offset)
	require.Equal(t, "db", *schemaProvider.lastFilter.DatabaseName)
	require.Equal(t, "sch", *schemaProvider.lastFilter.SchemaName)
	require.Equal(t, "tbl", *schemaProvider.lastFilter.TableName)
	require.Equal(t, "open", *schemaProvider.lastFilter.Status)
}

func TestGetColumnMismatchGroup(t *testing.T) {
	handler, _, _ := setupHandlerForTests()

	r := chi.NewRouter()
	r.Get("/api/column-mismatch-groups/{id}", handler.GetColumnMismatchGroup)

	req := httptest.NewRequest(http.MethodGet, "/api/column-mismatch-groups/1", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var resp models.ColumnMismatchGroupWithItems
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	require.Equal(t, int64(1), resp.Group.ID)

	notFoundReq := httptest.NewRequest(http.MethodGet, "/api/column-mismatch-groups/999", nil)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, notFoundReq)
	require.Equal(t, http.StatusNotFound, rr.Code)
}

func TestApplyColumnMismatchGroup(t *testing.T) {
	handler, _, _ := setupHandlerForTests()

	r := chi.NewRouter()
	r.Post("/api/column-mismatch-groups/{id}/apply", handler.ApplyColumnMismatchGroup)

	body := bytes.NewBufferString(`{"ignores":["col1"]}`)
	req := httptest.NewRequest(http.MethodPost, "/api/column-mismatch-groups/1/apply", body)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]string
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	require.Equal(t, "ok", resp["status"])

	badReq := httptest.NewRequest(http.MethodPost, "/api/column-mismatch-groups/abc/apply", nil)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, badReq)
	require.Equal(t, http.StatusBadRequest, rr.Code)

	emptyReq := httptest.NewRequest(http.MethodPost, "/api/column-mismatch-groups/1/apply", bytes.NewBufferString("bad"))
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, emptyReq)
	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func ptr[T any](v T) *T { return &v }
