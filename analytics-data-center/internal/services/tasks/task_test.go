package tasksserivce

import (
	"context"
	"testing"

	"analyticDataCenter/analytics-data-center/internal/domain/models"
	loggerpkg "analyticDataCenter/analytics-data-center/internal/logger"

	"github.com/stretchr/testify/require"
)

type mockSysDB struct {
	createCalled  bool
	createTaskID  string
	createStatus  string
	changeCalled  bool
	changeTaskID  string
	changeStatus  string
	changeComment string
	getCalled     bool
	getTaskID     string
	taskResult    models.Task
	taskErr       error
	createErr     error
	changeErr     error
}

func (m *mockSysDB) CreateTask(ctx context.Context, taskID, status string) error {
	m.createCalled = true
	m.createTaskID = taskID
	m.createStatus = status
	return m.createErr
}

func (m *mockSysDB) GetTask(ctx context.Context, taskID string) (models.Task, error) {
	m.getCalled = true
	m.getTaskID = taskID
	return m.taskResult, m.taskErr
}

func (m *mockSysDB) ChangeStatusTask(ctx context.Context, taskID, newStatus, comment string) error {
	m.changeCalled = true
	m.changeTaskID = taskID
	m.changeStatus = newStatus
	m.changeComment = comment
	return m.changeErr
}

// Unused SchemaProvider methods
func (m *mockSysDB) GetView(ctx context.Context, idView int64) (models.View, error) {
	return models.View{}, nil
}
func (m *mockSysDB) GetSchems(ctx context.Context, source, schema, table string) ([]int, error) {
	return nil, nil
}
func (m *mockSysDB) UpdateView(ctx context.Context, view models.View, schemaId int) error { return nil }

func (m *mockSysDB) UploadView(ctx context.Context, view models.View) (int64, error) { return 123, nil }
func (m *mockSysDB) ListTopics(ctx context.Context) ([]string, error)                { return nil, nil }
func (m *mockSysDB) GetTasks(ctx context.Context, filters models.TaskFilter) (tasks []models.Task, err error) {
	return []models.Task{}, nil
}
func (m *mockSysDB) CreateSuggestion(ctx context.Context, s models.ColumnRenameSuggestion) error {
	return nil
}
func (m *mockSysDB) ListSuggestions(ctx context.Context, filter models.ColumnRenameSuggestionFilter) ([]models.ColumnRenameSuggestion, error) {
	return nil, nil
}
func (m *mockSysDB) HasSuggestion(ctx context.Context, schemaID int64, database, schema, table string) (bool, error) {
	return false, nil
}
func (m *mockSysDB) GetSuggestionByID(ctx context.Context, id int64) (models.ColumnRenameSuggestion, error) {
	return models.ColumnRenameSuggestion{}, nil
}
func (m *mockSysDB) DeleteSuggestionByID(ctx context.Context, id int64) error { return nil }
func testLogger() *loggerpkg.Logger {
	return loggerpkg.New("test", "ru")
}
func (m *mockSysDB) CreateSuggestion(ctx context.Context, s models.ColumnRenameSuggestion) error {
	return nil
}
func (m *mockSysDB) ListSuggestions(ctx context.Context, filter models.ColumnRenameSuggestionFilter) ([]models.ColumnRenameSuggestion, error) {
	return []models.ColumnRenameSuggestion{}, nil
}
func (m *mockSysDB) HasSuggestion(ctx context.Context, schemaID int64, database, schema, table string) (bool, error) {
	return true, nil
}

func TestCreateTaskValidation(t *testing.T) {
	db := &mockSysDB{}
	svc := New(testLogger(), db, []string{"In progress", "Execution error", "Completed"})

	err := svc.CreateTask(context.Background(), "", "In progress")
	require.Error(t, err)

	err = svc.CreateTask(context.Background(), "task", "")
	require.Error(t, err)

	err = svc.CreateTask(context.Background(), "task", "Unknown")
	require.Error(t, err)
}

func TestCreateTaskSuccess(t *testing.T) {
	db := &mockSysDB{}
	svc := New(testLogger(), db, []string{"In progress", "Execution error", "Completed"})

	err := svc.CreateTask(context.Background(), "task123", "In progress")
	require.NoError(t, err)
	require.True(t, db.createCalled)
	require.Equal(t, "task123", db.createTaskID)
	require.Equal(t, "In progress", db.createStatus)
}

func TestChangeStatusTaskValidation(t *testing.T) {
	db := &mockSysDB{}
	svc := New(testLogger(), db, []string{"In progress", "Execution error", "Completed"})

	err := svc.ChangeStatusTask(context.Background(), "", "Completed", "")
	require.Error(t, err)

	err = svc.ChangeStatusTask(context.Background(), "id", "", "")
	require.Error(t, err)

	err = svc.ChangeStatusTask(context.Background(), "id", "Bad", "")
	require.Error(t, err)
}

func TestChangeStatusTaskSuccess(t *testing.T) {
	db := &mockSysDB{}
	svc := New(testLogger(), db, []string{"In progress", "Execution error", "Completed"})

	err := svc.ChangeStatusTask(context.Background(), "id1", "Completed", "done")
	require.NoError(t, err)
	require.True(t, db.changeCalled)
	require.Equal(t, "id1", db.changeTaskID)
	require.Equal(t, "Completed", db.changeStatus)
	require.Equal(t, "done", db.changeComment)
}

func TestGetTask(t *testing.T) {
	db := &mockSysDB{taskResult: models.Task{ID: "42", Status: "In progress"}}
	svc := New(testLogger(), db, []string{"In progress", "Execution error", "Completed"})

	task, err := svc.GetTask(context.Background(), "42")
	require.NoError(t, err)
	require.True(t, db.getCalled)
	require.Equal(t, "42", db.getTaskID)
	require.Equal(t, "42", task.ID)
	require.Equal(t, "In progress", task.Status)
}
