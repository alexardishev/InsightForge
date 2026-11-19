package serviceanalytics

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"analyticDataCenter/analytics-data-center/internal/domain/models"
	smtpsender "analyticDataCenter/analytics-data-center/internal/services/smtrsender"
	"analyticDataCenter/analytics-data-center/internal/storage"

	"github.com/stretchr/testify/require"
)

type etlTestTaskService struct {
	TaskService
	createTask   func(string) error
	changeStatus func(taskID, status, comment string) error
}

func (s etlTestTaskService) CreateTask(ctx context.Context, taskID, status string) error {
	if s.createTask != nil {
		return s.createTask(taskID)
	}
	return s.TaskService.CreateTask(ctx, taskID, status)
}

func (s etlTestTaskService) ChangeStatusTask(_ context.Context, taskID, status, comment string) error {
	return s.changeStatus(taskID, status, comment)
}

type etlTestNotifier struct {
	events    []models.Notification
	onPublish func(models.Notification)
}

func (n *etlTestNotifier) Publish(event models.Notification) {
	if n.onPublish != nil {
		n.onPublish(event)
		return
	}
	n.events = append(n.events, event)
}

func TestETLWorkerReportsFinalStatus(t *testing.T) {
	stageErr := errors.New("stage failed")
	cleanupErr := errors.New("cleanup failed")
	tests := []struct {
		name                   string
		countErr               error
		insertErr              error
		mergeErr               error
		indexErr               error
		replicaErr             error
		cleanupErr             error
		saveStatusErr          error
		status                 string
		reachedReplicaIdentity bool
	}{
		{name: "success", status: Completed, reachedReplicaIdentity: true},
		{name: "count fails", countErr: stageErr, status: Error},
		{name: "insert fails", insertErr: stageErr, status: Error},
		{name: "merge and cleanup fail", mergeErr: stageErr, cleanupErr: cleanupErr, status: Error},
		{name: "cleanup fails after successful load", cleanupErr: cleanupErr, status: Completed, reachedReplicaIdentity: true},
		{name: "index creation fails", indexErr: stageErr, status: Error},
		{name: "replica identity fails", replicaErr: stageErr, status: Error, reachedReplicaIdentity: true},
		{name: "status cannot be saved", saveStatusErr: stageErr, status: Completed, reachedReplicaIdentity: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			replicaIdentitySet := make(chan struct{}, 1)
			dwh := &mockDWH{
				defaultColumns: []string{"id"},
				insertErr:      tt.insertErr,
				mergeErr:       tt.mergeErr,
				indexErr:       tt.indexErr,
				onReplica: func(context.Context, string) error {
					replicaIdentitySet <- struct{}{}
					return tt.replicaErr
				},
			}
			oltp := &mockOLTP{
				countResult:  1,
				countErr:     tt.countErr,
				selectResult: []map[string]interface{}{{"id": 1}},
				indexResult: models.Indexes{Indexes: []models.Index{
					{IndexName: "idx", IndexDef: "CREATE INDEX idx ON public.users(id)"},
				}},
			}
			view := models.View{Name: "warehouse_users", Sources: []models.Source{{
				Name: "db1",
				Schemas: []models.Schema{{Name: "public", Tables: []models.Table{{
					Name:    "users",
					Columns: []models.Column{{Name: "id", Type: "int", IsPrimaryKey: true}},
				}}}},
			}}}
			schema := &mockSchemaProvider{views: map[int]models.View{1: view}}
			type statusUpdate struct {
				taskID, status, comment string
				afterReplicaIdentity    bool
			}
			statusUpdates := make(chan statusUpdate, 4)
			tasks := etlTestTaskService{TaskService: schema, changeStatus: func(taskID, status, comment string) error {
				update := statusUpdate{taskID: taskID, status: status, comment: comment}
				select {
				case <-replicaIdentitySet:
					update.afterReplicaIdentity = true
				default:
				}
				statusUpdates <- update
				return tt.saveStatusErr
			}}
			notifier := &etlTestNotifier{}
			svc := &AnalyticsDataCenterService{
				log:            getTestLogger(),
				SchemaProvider: schema,
				TaskService:    tasks,
				DWHProvider:    dwh,
				DWHDbName:      DbPostgres,
				OLTPDbName:     DbPostgres,
				OLTPFactory:    &mockFactory{store: map[string]storage.OLTPDB{"db1": oltp}},
				jobQueue:       make(chan TaskETL, 1),
				queueSlots:     make(chan struct{}, 1),
				notifier:       notifier,
			}
			taskID, err := svc.StartETLProcess(context.Background(), 1)
			require.NoError(t, err)
			tempTable := "temp_" + strings.ReplaceAll(taskID, "-", "") + "_1"
			if tt.cleanupErr != nil {
				dwh.deleteErrors = map[string]error{tempTable: tt.cleanupErr}
			}
			close(svc.jobQueue)
			svc.etlWorker()

			require.Len(t, statusUpdates, 1)
			update := <-statusUpdates
			require.Equal(t, taskID, update.taskID)
			require.Equal(t, tt.status, update.status)
			if tt.saveStatusErr != nil {
				require.Empty(t, notifier.events)
			} else {
				require.Len(t, notifier.events, 1)
				require.Equal(t, taskID, notifier.events[0].TaskID)
				require.Equal(t, tt.status, notifier.events[0].Status)
				require.Equal(t, update.comment, notifier.events[0].Message)
			}
			require.True(t, svc.reserveView(1), "a finished job must release the view even if the status write failed")
			require.Equal(t, tt.reachedReplicaIdentity, update.afterReplicaIdentity)
			if tt.countErr != nil || tt.mergeErr != nil || tt.indexErr != nil || tt.replicaErr != nil {
				require.Contains(t, update.comment, stageErr.Error())
			}
			if tt.cleanupErr != nil {
				require.Contains(t, update.comment, cleanupErr.Error())
				require.Contains(t, update.comment, tempTable)
			}
			if tt.status == Completed {
				require.Contains(t, update.comment, CompletedTask)
			}
			require.Equal(t, []string{tempTable}, dwh.deleteCalls)
		})
	}
}

func TestStartETLRejectsFullQueueBeforeCreatingTask(t *testing.T) {
	createErr := errors.New("create task failed")
	createCalls := 0
	tasks := etlTestTaskService{createTask: func(string) error {
		createCalls++
		if createCalls == 1 {
			return createErr
		}
		return nil
	}}
	svc := &AnalyticsDataCenterService{
		TaskService: tasks,
		jobQueue:    make(chan TaskETL, 1),
		queueSlots:  make(chan struct{}, 1),
	}
	ctx := context.Background()
	_, err := svc.StartETLProcess(ctx, 1)
	require.ErrorIs(t, err, createErr)
	taskID, err := svc.StartETLProcess(ctx, 1)
	require.NoError(t, err)
	require.NotEmpty(t, taskID)

	result := make(chan error, 1)
	go func() {
		_, err := svc.StartETLProcess(ctx, 2)
		result <- err
	}()
	select {
	case err := <-result:
		require.ErrorIs(t, err, ErrETLQueueFull)
	case <-time.After(3 * time.Second):
		<-svc.jobQueue
		t.Fatal("StartETLProcess blocked on a full queue")
	}
	require.Len(t, svc.jobQueue, 1)
	require.Equal(t, 2, createCalls)
}

func TestETLWorkerRejectsDuplicateViewWithoutBlockingOthers(t *testing.T) {
	firstBlocked := make(chan struct{})
	releaseFirst := make(chan struct{})
	unblock := sync.OnceFunc(func() { close(releaseFirst) })
	defer unblock()
	blockFirst := sync.OnceFunc(func() {
		close(firstBlocked)
		<-releaseFirst
	})
	dwh := &mockDWH{
		defaultColumns: []string{"id"},
		onReplica: func(_ context.Context, table string) error {
			if table == "first_view" {
				blockFirst()
			}
			return nil
		},
	}
	oltp := &mockOLTP{countResult: 1, selectResult: []map[string]interface{}{{"id": 1}}}
	source := models.Source{Name: "db1", Schemas: []models.Schema{{Name: "public", Tables: []models.Table{{
		Name: "users", Columns: []models.Column{{Name: "id", Type: "int", IsPrimaryKey: true}},
	}}}}}
	schema := &mockSchemaProvider{views: map[int]models.View{
		1: {Name: "first_view", Sources: []models.Source{source}},
		2: {Name: "second_view", Sources: []models.Source{source}},
	}}
	statusWriteStarted := make(chan struct{})
	statusWriteRelease := make(chan struct{})
	finishStatusWrite := sync.OnceFunc(func() { close(statusWriteRelease) })
	defer finishStatusWrite()
	firstSubmitted := make(chan struct{})
	var first string
	tasks := etlTestTaskService{TaskService: schema, changeStatus: func(taskID, _, _ string) error {
		<-firstSubmitted
		if taskID == first {
			close(statusWriteStarted)
			<-statusWriteRelease
		}
		return nil
	}}
	factory := &mockFactory{store: map[string]storage.OLTPDB{"db1": oltp}}
	svc := New(getTestLogger(), schema, tasks, dwh, factory, DbPostgres, "", DbPostgres, false, smtpsender.SMTP{})
	completed := make(chan models.Notification, 3)
	svc.SetNotifier(&etlTestNotifier{onPublish: func(event models.Notification) { completed <- event }})
	defer close(svc.jobQueue)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	var err error
	first, err = svc.StartETLProcess(ctx, 1)
	close(firstSubmitted)
	require.NoError(t, err)
	select {
	case <-firstBlocked:
	case <-ctx.Done():
		t.Fatal("first job did not reach replica identity")
	}

	_, err = svc.StartETLProcess(ctx, 1)
	require.ErrorIs(t, err, ErrETLViewInProgress)
	require.Empty(t, svc.jobQueue)

	second, err := svc.StartETLProcess(ctx, 2)
	require.NoError(t, err)
	select {
	case update := <-completed:
		require.Equal(t, second, update.TaskID)
		require.Equal(t, Completed, update.Status, update.Message)
	case <-ctx.Done():
		t.Fatal("second job did not complete while first job was blocked")
	}
	unblock()
	select {
	case <-statusWriteStarted:
	case <-ctx.Done():
		t.Fatal("first job did not reach the final status write")
	}
	_, err = svc.StartETLProcess(ctx, 1)
	require.ErrorIs(t, err, ErrETLViewInProgress)
	finishStatusWrite()
	select {
	case update := <-completed:
		require.Equal(t, first, update.TaskID)
		require.Equal(t, Completed, update.Status, update.Message)
	case <-ctx.Done():
		t.Fatal("first job did not complete after it was released")
	}

	restarted, err := svc.StartETLProcess(ctx, 1)
	require.NoError(t, err)
	require.NotEqual(t, first, restarted)
	select {
	case update := <-completed:
		require.Equal(t, restarted, update.TaskID)
		require.Equal(t, Completed, update.Status, update.Message)
	case <-ctx.Done():
		t.Fatal("restarted job did not complete after the first job finished")
	}
	require.Len(t, dwh.createCalls, 3)
	require.NotEqual(t, dwh.createCalls[0], dwh.createCalls[1])
	require.Equal(t, []string{dwh.createCalls[1], dwh.createCalls[0], dwh.createCalls[2]}, dwh.deleteCalls)
}
