package serviceanalytics

import (
	"analyticDataCenter/analytics-data-center/internal/domain/models"
	sqlgenerator "analyticDataCenter/analytics-data-center/internal/lib/SQLGenerator"
	"analyticDataCenter/analytics-data-center/internal/storage"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

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
)

type TaskETL struct {
	ViewID int64
	TaskID string
}

type AnalyticsDataCenterService struct {
	log            *slog.Logger
	SchemaProvider storage.SysDB
	TaskService    TaskService
	DWHProvider    storage.DWHDB
	OLTPFactory    storage.OLTPFactory
	DWHDbName      string
	OLTPDbName     string
	jobQueue       chan TaskETL
}

type TaskService interface {
	CreateTask(ctx context.Context, taskID string, status string) error
	GetTask(ctx context.Context, taskID string) (models.Task, error)
	ChangeStatusTask(ctx context.Context, taskID string, status string, comment string) error
}

func New(
	log *slog.Logger,
	schemaProvider storage.SysDB,
	taskService TaskService,
	dwhProvider storage.DWHDB,
	OLTPFactory storage.OLTPFactory,
	DWHDbName string,
	OLTPDbName string,

) *AnalyticsDataCenterService {
	service := &AnalyticsDataCenterService{
		log:            log,
		SchemaProvider: schemaProvider,
		TaskService:    taskService,
		DWHProvider:    dwhProvider,
		OLTPFactory:    OLTPFactory,
		DWHDbName:      DWHDbName,
		OLTPDbName:     OLTPDbName,
		jobQueue:       make(chan TaskETL, 100),
	}
	go service.etlWorker()
	return service
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

func (a *AnalyticsDataCenterService) etlWorker() {
	for job := range a.jobQueue {
		log := a.log.With(
			slog.String("task", job.TaskID),
			slog.String("component", "ETLWorker"),
		)
		log.Info("Начало обработки задачи")

		ctx := context.Background()
		err := a.runETL(ctx, job.ViewID, job.TaskID)
		if err != nil {
			log.Error("Ошибка при выполнении ETL", slog.String("error", err.Error()))
			a.TaskService.ChangeStatusTask(ctx, job.TaskID, Error, err.Error())
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
	log.Info("ETL start")
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
	if a.OLTPDbName == DbPostgres {
		// Переделать на вызов вспомогательной функции, здесь должен быть чистый код без условий
		queries, duplicates, err := sqlgenerator.GenerateQueryCreateTempTablePostgres(&viewSchema, log)
		if err != nil {
			log.Error("не удалось сгенерировать запросы генератором SQL", slog.String("error", err.Error()))
			a.TaskService.ChangeStatusTask(ctx, taskID, Error, ErrorCreateTemplateTable)
			return fmt.Errorf("%s:%s", op, err)
		}

		queriesInit = queries

		if len(duplicates) > 0 {
			log.Warn("duplicates", slog.String("duplicates", strings.Join(duplicates, ",")))

		}

	}
	for _, tempTable := range queriesInit.Queries {
		tempTables = append(tempTables, tempTable.TableName)
		log.Info("Временная таблица", slog.String("Таблица", tempTable.TableName))
	}
	err = a.createTempTables(ctx, queriesInit)
	if err != nil {
		log.Error("не удалось создать временные таблицы", slog.String("error", err.Error()))
		a.TaskService.ChangeStatusTask(ctx, taskID, Error, ErrorCreateTemplateTable)
		return fmt.Errorf("%s:%s", op, err)
	}

	countInsertData, err := a.getCountInsertData(ctx, viewSchema, tempTables)
	if err != nil {
		log.Error("не удалось получить количество", slog.String("error", err.Error()))
		a.TaskService.ChangeStatusTask(ctx, taskID, Error, ErrorCountInsertData)
		return fmt.Errorf("%s:%s", op, err)
	}
	backgroundCtx := context.Background() // независимый от вызова клиента

	go func() {
		ok, err := a.prepareAndInsertData(backgroundCtx, &countInsertData, &viewSchema)
		if err != nil {
			log.Error("не удалось получить данные для вставки", slog.String("error", err.Error()))
			a.TaskService.ChangeStatusTask(ctx, taskID, Error, ErrorSelectInsertData)
			// return "", fmt.Errorf("%s:%s", op, err)
		}
		fmt.Println(ok)
	}()

	log.Info("количество записей в таблице -", slog.Any("slice", countInsertData))
	return nil
}
