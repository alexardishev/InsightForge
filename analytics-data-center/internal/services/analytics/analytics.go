package serviceanalytics

import (
	"analyticDataCenter/analytics-data-center/internal/domain/models"
	sqlgenerator "analyticDataCenter/analytics-data-center/internal/lib/SQLGenerator"
	"analyticDataCenter/analytics-data-center/internal/notifications"
	smtpsender "analyticDataCenter/analytics-data-center/internal/services/smtrsender"
	"analyticDataCenter/analytics-data-center/internal/storage"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	loggerpkg "analyticDataCenter/analytics-data-center/internal/logger"

	"github.com/google/uuid"
)

var (
	ErrInvalidSchemID = errors.New("invalid schemaID")
)

const (
	DbPostgres   = "postgres"
	DbClickhouse = "clickhouse"
)

const (
	Progress  = "In progress"
	Error     = "Execution error"
	Completed = "Completed"
)

const (
	ErrorCreateTemplateTable = "Не удалось создать временные таблицы"
	ErrorCountInsertData     = "Не удалось посчитать количество вставляемых данных"
	ErrorSelectInsertData    = "Не удалось получить данный для вставки"
	ErrorReplicaFullData     = "Не удалось включиь полную репликацию таблицы"
	CompletedTask            = "Задача завершена успешно"
)

type TaskETL struct {
	ViewID int64
	TaskID string
}

// TopicNotifier allows to send information about new topics to subscribe.
type TopicNotifier interface {
	EnqueueTopic(topic string)
}

type AnalyticsDataCenterService struct {
	log                     *loggerpkg.Logger
	SchemaProvider          storage.SysDB
	RenameSuggestionStorage storage.ColumnRenameSuggestionStorage
	ColumnMismatchStorage   storage.ColumnMismatchStorage
	TaskService             TaskService
	DWHProvider             storage.DWHDB
	OLTPFactory             storage.OLTPFactory
	DWHDbName               string
	DWHDbPath               string
	OLTPDbName              string
	RenameHeuristicEnabled  bool
	jobQueue                chan TaskETL
	eventQueue              chan models.CDCEvent
	SMTPClient              smtpsender.SMTP
	topicNotifier           TopicNotifier
	notifier                notifications.Notifier
}

type TaskService interface {
	CreateTask(ctx context.Context, taskID string, status string) error
	GetTask(ctx context.Context, taskID string) (models.Task, error)
	ChangeStatusTask(ctx context.Context, taskID string, status string, comment string) error
	GetTasks(ctx context.Context, taskFilter models.TaskFilter) ([]models.Task, error)
}

func New(
	log *loggerpkg.Logger,
	schemaProvider storage.SysDB,
	taskService TaskService,
	dwhProvider storage.DWHDB,
	OLTPFactory storage.OLTPFactory,
	DWHDbName string,
	DWHDbPath string,
	OLTPDbName string,
	renameHeuristic bool,
	SMTPClient smtpsender.SMTP,

) *AnalyticsDataCenterService {
	service := &AnalyticsDataCenterService{
		log:                     log,
		SchemaProvider:          schemaProvider,
		RenameSuggestionStorage: schemaProvider,
		ColumnMismatchStorage:   schemaProvider,
		TaskService:             taskService,
		DWHProvider:             dwhProvider,
		OLTPFactory:             OLTPFactory,
		DWHDbName:               DWHDbName,
		DWHDbPath:               DWHDbPath,
		OLTPDbName:              OLTPDbName,
		RenameHeuristicEnabled:  renameHeuristic,
		jobQueue:                make(chan TaskETL, 100),
		eventQueue:              make(chan models.CDCEvent, 100),
		SMTPClient:              SMTPClient,
	}
	go service.etlWorker()
	go service.eventWorker()
	return service
}

// SetTopicNotifier injects notifier implementation used to send topic updates.
func (a *AnalyticsDataCenterService) SetTopicNotifier(n TopicNotifier) {
	a.topicNotifier = n
}

// SetNotifier injects websocket notification publisher.
func (a *AnalyticsDataCenterService) SetNotifier(notifier notifications.Notifier) {
	a.notifier = notifier
}

func (a *AnalyticsDataCenterService) StartETLProcess(ctx context.Context, idView int64) (taskID string, err error) {
	taskID = uuid.NewString()

	err = a.TaskService.CreateTask(ctx, taskID, Progress)
	if err != nil {
		return "", err
	}

	a.jobQueue <- TaskETL{
		TaskID: taskID,
		ViewID: idView,
	}
	return taskID, nil

}

func (a *AnalyticsDataCenterService) ListViews(ctx context.Context) ([]models.SchemaInfo, error) {
	views, err := a.SchemaProvider.ListViews(ctx)
	if err != nil {
		return nil, err
	}
	return views, nil
}

func (a *AnalyticsDataCenterService) etlWorker() {
	for job := range a.jobQueue {
		log := a.log.With(
			slog.String("task", job.TaskID),
			slog.String("component", "ETLWorker"),
		)
		log.InfoMsg(loggerpkg.MsgETLWorkerStart)

		ctx := context.Background()
		err := a.runETL(ctx, job.ViewID, job.TaskID)
		if err != nil {
			log.ErrorMsg(loggerpkg.MsgInsertDataFailed, slog.String("error", err.Error()))
			a.TaskService.ChangeStatusTask(ctx, job.TaskID, Error, err.Error())
			a.notifyTaskStatus(job.TaskID, Error, err.Error())
		}
	}
}

func (a *AnalyticsDataCenterService) runETL(ctx context.Context, idView int64, taskID string) error {
	const op = "analytics.StartETLProcess"
	var queriesInit models.Queries
	var tempTables []string
	log := a.log.With(
		slog.String("op", op),
		slog.Int64("idSchema", idView),
	)
	log.InfoMsg(loggerpkg.MsgETLStart)
	viewSchema, err := a.SchemaProvider.GetView(ctx, idView)
	if err != nil {
		if errors.Is(err, storage.ErrSchemaNotFound) {
			a.log.Warn("view not found", slog.String("error", err.Error()))
			a.TaskService.ChangeStatusTask(ctx, taskID, Error, ErrorCreateTemplateTable)
			return fmt.Errorf("%s:%s", op, ErrInvalidSchemID)

		}
		a.TaskService.ChangeStatusTask(ctx, taskID, Error, ErrorCreateTemplateTable)
		a.log.Warn("view not found", slog.String("error", err.Error()))
		return fmt.Errorf("%s:%s", op, err)
	}
	queries, duplicates, err := sqlgenerator.GenerateQueryCreateTempTable(&viewSchema, log.Logger, a.DWHDbName)
	if err != nil {
		log.ErrorMsg(loggerpkg.MsgGenerateQueriesFailed, slog.String("error", err.Error()))
		a.TaskService.ChangeStatusTask(ctx, taskID, Error, ErrorCreateTemplateTable)
		return fmt.Errorf("%s:%s", op, err)
	}

	queriesInit = queries

	if len(duplicates) > 0 {
		log.Warn("duplicates", slog.String("duplicates", strings.Join(duplicates, ",")))

	}
	for _, tempTable := range queriesInit.Queries {
		tempTables = append(tempTables, tempTable.TableName)
		log.InfoMsg(loggerpkg.MsgTempTable, slog.String("table", tempTable.TableName))
	}
	err = a.createTempTables(ctx, queriesInit)
	if err != nil {
		log.ErrorMsg(loggerpkg.MsgCreateTempTablesFailed, slog.String("error", err.Error()))
		a.TaskService.ChangeStatusTask(ctx, taskID, Error, ErrorCreateTemplateTable)
		return fmt.Errorf("%s:%s", op, err)
	}

	countInsertData, err := a.getCountInsertData(ctx, viewSchema, tempTables)
	if err != nil {
		log.ErrorMsg(loggerpkg.MsgCountRowsFailed, slog.String("error", err.Error()))
		a.TaskService.ChangeStatusTask(ctx, taskID, Error, ErrorCountInsertData)
		return fmt.Errorf("%s:%s", op, err)
	}
	backgroundCtx := context.Background() // независимый от вызова клиента

	go func() {
		_, err := a.prepareAndInsertData(backgroundCtx, &countInsertData, &viewSchema)
		if err != nil {
			log.ErrorMsg(loggerpkg.MsgInsertDataFailed, slog.String("error", err.Error()))
			a.TaskService.ChangeStatusTask(ctx, taskID, Error, ErrorSelectInsertData)
			// return "", fmt.Errorf("%s:%s", op, err)
		}

		err = a.transferIndixesAndConstraint(ctx, &viewSchema, a.DWHDbName)
		if err != nil {
			log.ErrorMsg(loggerpkg.MsgTransferIndexesFailed, slog.String("error", err.Error()))
			a.TaskService.ChangeStatusTask(ctx, taskID, Error, ErrorSelectInsertData)
		}
		//Нужно для постгри, если DWH и OLTP одна БД.
		if a.DWHDbName == DbPostgres {
			err = a.DWHProvider.ReplicaIdentityFull(ctx, strings.ToLower(viewSchema.Name))
		}
		if err != nil {
			log.ErrorMsg(loggerpkg.MsgEnableReplicationFailed, slog.String("error", err.Error()))
			a.TaskService.ChangeStatusTask(ctx, taskID, Error, ErrorSelectInsertData)
		}
		log.InfoMsg(loggerpkg.MsgReplicationEnabled)

	}()

	log.InfoMsg(loggerpkg.MsgTableRecordCount, slog.Any("count", countInsertData))
	a.TaskService.ChangeStatusTask(ctx, taskID, Completed, CompletedTask)
	a.notifyTaskStatus(taskID, Completed, CompletedTask)
	return nil
}

func (a *AnalyticsDataCenterService) notifyTaskStatus(taskID, status, message string) {
	if a.notifier == nil {
		return
	}

	notification := models.Notification{
		Type:      "task_status",
		Title:     "ETL задача",
		Message:   message,
		TaskID:    taskID,
		Status:    status,
		CreatedAt: time.Now(),
	}

	a.notifier.Publish(notification)
}
