package serviceanalytics

import (
	"context"
	"errors"
	"testing"

	"analyticDataCenter/analytics-data-center/internal/domain/models"
	"analyticDataCenter/analytics-data-center/internal/storage"

	"github.com/stretchr/testify/require"
)

type etlTestTaskService struct {
	TaskService
	changeStatus func(taskID, status, comment string) error
}

func (s etlTestTaskService) ChangeStatusTask(_ context.Context, taskID, status, comment string) error {
	return s.changeStatus(taskID, status, comment)
}

type etlTestNotifier struct {
	events []models.Notification
}

func (n *etlTestNotifier) Publish(event models.Notification) {
	n.events = append(n.events, event)
}

func TestETLWorkerReportsFinalStatus(t *testing.T) {
	stageErr := errors.New("stage failed")
	tests := []struct {
		name       string
		countErr   error
		insertErr  error
		mergeErr   error
		indexErr   error
		replicaErr error
		status     string
	}{
		{name: "success", status: Completed},
		{name: "count fails", countErr: stageErr, status: Error},
		{name: "insert fails", insertErr: stageErr, status: Error},
		{name: "merge and cleanup fail", mergeErr: stageErr, status: Error},
		{name: "index creation fails", indexErr: stageErr, status: Error},
		{name: "replica identity fails", replicaErr: stageErr, status: Error},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			const tempTable = "temp_db1_public_users"
			replicaIdentitySet := make(chan struct{}, 1)
			dwh := &mockDWH{
				columns:   map[string][]string{tempTable: {"id"}},
				insertErr: tt.insertErr,
				mergeErr:  tt.mergeErr,
				indexErr:  tt.indexErr,
				onReplica: func(context.Context, string) error {
					replicaIdentitySet <- struct{}{}
					return tt.replicaErr
				},
			}
			if tt.mergeErr != nil {
				dwh.deleteErrors = map[string]error{tempTable: errors.New("cleanup failed")}
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
				return nil
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
				notifier:       notifier,
			}
			taskID, err := svc.StartETLProcess(context.Background(), 1)
			require.NoError(t, err)
			close(svc.jobQueue)
			svc.etlWorker()

			require.Len(t, statusUpdates, 1)
			update := <-statusUpdates
			require.Equal(t, taskID, update.taskID)
			require.Equal(t, tt.status, update.status)
			require.Len(t, notifier.events, 1)
			require.Equal(t, taskID, notifier.events[0].TaskID)
			require.Equal(t, tt.status, notifier.events[0].Status)
			require.Equal(t, update.comment, notifier.events[0].Message)
			require.Equal(t, tt.status == Completed || tt.replicaErr != nil, update.afterReplicaIdentity)
			if tt.status == Error && tt.insertErr == nil {
				require.Contains(t, update.comment, stageErr.Error())
			}
			require.Equal(t, []string{tempTable}, dwh.deleteCalls)
		})
	}
}
