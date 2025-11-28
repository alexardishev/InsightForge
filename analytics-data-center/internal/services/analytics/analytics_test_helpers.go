package serviceanalytics

import (
	"context"
	"fmt"

	loggerpkg "analyticDataCenter/analytics-data-center/internal/logger"

	"analyticDataCenter/analytics-data-center/internal/domain/models"
	"analyticDataCenter/analytics-data-center/internal/storage"
)

func getTestLogger() *loggerpkg.Logger {
	return loggerpkg.New("test", "ru")
}

type mockDWH struct {
	createCalls  []string
	deleteCalls  []string
	columns      map[string][]string
	createErrors map[string]error
	deleteErrors map[string]error
	insertCalls  []string
	insertErr    error
	mergeCalls   int
	mergeErr     error
	indexCalls   []string
	indexErr     error
}

func (m *mockDWH) CreateTempTable(_ context.Context, _ string, name string) error {
	m.createCalls = append(m.createCalls, name)
	if err, ok := m.createErrors[name]; ok {
		return err
	}
	return nil
}

func (m *mockDWH) DeleteTempTable(_ context.Context, name string) error {
	m.deleteCalls = append(m.deleteCalls, name)
	if err, ok := m.deleteErrors[name]; ok {
		return err
	}
	return nil
}

func (m *mockDWH) CreateIndex(_ context.Context, query string) error {
	m.indexCalls = append(m.indexCalls, query)
	return m.indexErr
}
func (m *mockDWH) CreateConstraint(_ context.Context, _ string) error { return nil }
func (m *mockDWH) InsertDataToDWH(_ context.Context, query string) error {
	m.insertCalls = append(m.insertCalls, query)
	return m.insertErr
}
func (m *mockDWH) RenameColumn(context.Context, string) error { return nil }
func (m *mockDWH) GetColumnsTables(_ context.Context, _ string, table string) ([]string, error) {
	if cols, ok := m.columns[table]; ok {
		return cols, nil
	}
	return nil, fmt.Errorf("no columns")
}
func (m *mockDWH) MergeTempTables(_ context.Context, _ string) error {
	m.mergeCalls++
	return m.mergeErr
}
func (m *mockDWH) ReplicaIdentityFull(context.Context, string) error { return nil }
func (m *mockDWH) InsertOrUpdateTransactional(context.Context, string, map[string]interface{}, []string) error {
	return nil
}

// ---- OLTP mocks ----
type mockOLTP struct {
	countResult  int64
	countErr     error
	selectResult []map[string]interface{}
	selectErr    error
	indexResult  models.Indexes
	indexErr     error
}

func (m *mockOLTP) GetCountInsertData(context.Context, string) (int64, error) {
	if m.countErr != nil {
		return 0, m.countErr
	}
	return m.countResult, nil
}
func (m *mockOLTP) SelectDataToInsert(context.Context, string) ([]map[string]interface{}, error) {
	if m.selectErr != nil {
		return nil, m.selectErr
	}
	return m.selectResult, nil
}
func (m *mockOLTP) GetIndexes(context.Context, string, string) (models.Indexes, error) {
	if m.indexErr != nil {
		return models.Indexes{}, m.indexErr
	}
	return m.indexResult, nil
}
func (m *mockOLTP) GetConstraint(context.Context, string, string) (models.Constraints, error) {
	return models.Constraints{}, nil
}
func (m *mockOLTP) GetColumns(context.Context, string, string) ([]models.Column, error) {
	return []models.Column{}, nil
}

func (m *mockOLTP) GetSchemas(context.Context, string) ([]models.Schema, error) {
	return []models.Schema{}, nil
}
func (m *mockOLTP) GetTables(context.Context, string) ([]models.Table, error) {
	return []models.Table{}, nil
}
func (m *mockOLTP) GetTablesPaginated(context.Context, string, int, int) ([]models.Table, error) {
	return []models.Table{}, nil
}

func (m *mockOLTP) GetColumnInfo(context.Context, string, string) (models.ColumnInfo, error) {
	return models.ColumnInfo{}, nil
}

// ---- factory ----
type mockFactory struct {
	store map[string]storage.OLTPDB
	err   error
}

func (m *mockFactory) GetOLTPStorage(_ context.Context, name string) (storage.OLTPDB, error) {
	if m.err != nil {
		return nil, m.err
	}
	st, ok := m.store[name]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return st, nil
}
func (m *mockFactory) CloseAll() error                                  { return nil }
func (m *mockFactory) GetOLTPStrings(context.Context) map[string]string { return nil }
