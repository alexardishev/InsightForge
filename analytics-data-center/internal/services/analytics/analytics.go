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
	"sync"
	"time"

	loggerpkg "analyticDataCenter/analytics-data-center/internal/logger"

	"github.com/google/uuid"
)

var (
	ErrInvalidSchemID    = errors.New("invalid schemaID")
	ErrETLQueueFull      = errors.New("очередь ETL заполнена")
	ErrETLViewInProgress = errors.New("ETL для этой витрины уже выполняется или находится в очереди")
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
	etlWorkerCount = 2
	etlQueueSize   = 100
)

const (
	ErrorCreateTemplateTable = "Не удалось создать временные таблицы"
	ErrorCountInsertData     = "Не удалось посчитать количество вставляемых данных"
	ErrorReplicaFullData     = "Не удалось включить полную репликацию таблицы"
	CompletedTask            = "Задача завершена успешно"
)

type TaskETL struct {
	ViewID int64
	TaskID uuid.UUID
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
	queueSlots              chan struct{}
	activeViewsMu           sync.Mutex
	activeViews             map[int64]struct{}
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
		jobQueue:                make(chan TaskETL, etlQueueSize),
		queueSlots:              make(chan struct{}, etlQueueSize),
		SMTPClient:              SMTPClient,
	}
	for range etlWorkerCount {
		go service.etlWorker()
	}
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

func (a *AnalyticsDataCenterService) StartETLProcess(ctx context.Context, idView int64) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	// Сначала резервируем место, чтобы задача в системной БД всегда попала в очередь.
	select {
	case a.queueSlots <- struct{}{}:
	default:
		return "", ErrETLQueueFull
	}
	if !a.reserveView(idView) {
		<-a.queueSlots
		return "", ErrETLViewInProgress
	}

	taskID := uuid.New()
	if err := a.TaskService.CreateTask(ctx, taskID.String(), Progress); err != nil {
		a.releaseView(idView)
		<-a.queueSlots
		return "", err
	}

	a.jobQueue <- TaskETL{
		TaskID: taskID,
		ViewID: idView,
	}
	return taskID.String(), nil
}

func (a *AnalyticsDataCenterService) reserveView(idView int64) bool {
	a.activeViewsMu.Lock()
	defer a.activeViewsMu.Unlock()
	if _, exists := a.activeViews[idView]; exists {
		return false
	}
	if a.activeViews == nil {
		a.activeViews = make(map[int64]struct{})
	}
	a.activeViews[idView] = struct{}{}
	return true
}

func (a *AnalyticsDataCenterService) releaseView(idView int64) {
	a.activeViewsMu.Lock()
	defer a.activeViewsMu.Unlock()
	delete(a.activeViews, idView)
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
		<-a.queueSlots
		a.processETLJob(job)
	}
}

func (a *AnalyticsDataCenterService) processETLJob(job TaskETL) {
	taskID := job.TaskID.String()
	log := a.log.With(
		slog.String("task", taskID),
		slog.String("component", "ETLWorker"),
	)
	log.InfoMsg(loggerpkg.MsgETLWorkerStart)

	ctx := context.Background()
	status, comment := Completed, CompletedTask
	warning, err := a.runETL(ctx, job)
	if err != nil {
		log.Error("задача ETL завершилась с ошибкой", slog.String("error", err.Error()))
		status, comment = Error, err.Error()
	} else if warning != "" {
		comment += ". Требуется ручная очистка. " + warning
	}
	statusErr := a.TaskService.ChangeStatusTask(ctx, taskID, status, comment)
	a.releaseView(job.ViewID)
	if statusErr != nil {
		log.ErrorMsg(loggerpkg.MsgChangeStatusFailed, slog.String("status", status), slog.String("error", statusErr.Error()))
		return
	}
	a.notifyTaskStatus(taskID, status, comment)
}

func (a *AnalyticsDataCenterService) runETL(ctx context.Context, job TaskETL) (warning string, err error) {
	const op = "analytics.StartETLProcess"
	log := a.log.With(
		slog.String("op", op),
		slog.Int64("idSchema", job.ViewID),
	)
	log.InfoMsg(loggerpkg.MsgETLStart)
	viewSchema, err := a.SchemaProvider.GetView(ctx, job.ViewID)
	if err != nil {
		log.Warn("не удалось получить схему витрины", slog.String("error", err.Error()))
		if errors.Is(err, storage.ErrSchemaNotFound) {
			return "", fmt.Errorf("%s: %w", op, ErrInvalidSchemID)
		}
		return "", fmt.Errorf("%s: %w", op, err)
	}
	queries, duplicates, err := sqlgenerator.GenerateQueryCreateTempTable(&viewSchema, log.Logger, a.DWHDbName, job.TaskID)
	if err != nil {
		log.ErrorMsg(loggerpkg.MsgGenerateQueriesFailed, slog.String("error", err.Error()))
		return "", fmt.Errorf("%s: %w", ErrorCreateTemplateTable, err)
	}

	if len(duplicates) > 0 {
		log.Warn("duplicates", slog.String("duplicates", strings.Join(duplicates, ",")))
	}
	tempTables := make([]string, 0, len(queries.Queries))
	for _, tempTable := range queries.Queries {
		tempTables = append(tempTables, tempTable.TableName)
		log.InfoMsg(loggerpkg.MsgTempTable, slog.String("table", tempTable.TableName))
	}
	if err := a.createTempTables(ctx, queries); err != nil {
		log.ErrorMsg(loggerpkg.MsgCreateTempTablesFailed, slog.String("error", err.Error()))
		return "", fmt.Errorf("%s: %w", ErrorCreateTemplateTable, err)
	}
	defer func() {
		if cleanupErr := a.DeleteTempTables(ctx, tempTables); cleanupErr != nil {
			if err != nil {
				err = errors.Join(err, cleanupErr)
			} else {
				warning = cleanupErr.Error()
			}
		}
	}()

	countInsertData, err := a.getCountInsertData(ctx, viewSchema, tempTables)
	if err != nil {
		log.ErrorMsg(loggerpkg.MsgCountRowsFailed, slog.String("error", err.Error()))
		return "", fmt.Errorf("%s: %w", ErrorCountInsertData, err)
	}
	if err := a.prepareAndInsertData(ctx, countInsertData, viewSchema); err != nil {
		return "", fmt.Errorf("не удалось загрузить данные в DWH: %w", err)
	}

	if err := a.transferIndixesAndConstraint(ctx, &viewSchema, a.DWHDbName); err != nil {
		log.ErrorMsg(loggerpkg.MsgTransferIndexesFailed, slog.String("error", err.Error()))
		return "", fmt.Errorf("не удалось перенести индексы: %w", err)
	}
	if a.DWHDbName == DbPostgres {
		if err := a.DWHProvider.ReplicaIdentityFull(ctx, strings.ToLower(viewSchema.Name)); err != nil {
			log.ErrorMsg(loggerpkg.MsgEnableReplicationFailed, slog.String("error", err.Error()))
			return "", fmt.Errorf("%s: %w", ErrorReplicaFullData, err)
		}
		log.InfoMsg(loggerpkg.MsgReplicationEnabled)
	}

	log.InfoMsg(loggerpkg.MsgTableRecordCount, slog.Any("count", countInsertData))
	return "", nil
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
