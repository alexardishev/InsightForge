package serviceanalytics

import (
	"analyticDataCenter/analytics-data-center/internal/domain/models"
	sqlgenerator "analyticDataCenter/analytics-data-center/internal/lib/SQLGenerator"
	"analyticDataCenter/analytics-data-center/internal/storage"
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
)

var (
	ErrInvalidSchemID = errors.New("invalid schemaID")
)

const (
	Progress  = "In progress"
	Error     = "Execution error"
	Completed = "Completed"
)

const (
	ErrorCreateTemplateTable = "Не удалось создать временные таблицы"
)

const (
	DbPostgres   = "postgres"
	DbClickhouse = "clickhouse"
)

type AnalyticsDataCenterService struct {
	log            *slog.Logger
	SchemaProvider SchemaProvider
	TaskService    TaskService
	TableProvider  TableProvider
	DWHDbName      string
	OLTPDbName     string
}

type SchemaProvider interface {
	GetView(ctx context.Context, idView int64) (models.View, error)
}

type TableProvider interface {
	CreateTempTablePostgres(ctx context.Context, query string) error
	DeleteTempTablePostgres(ctx context.Context, tableName string) error
}

type TaskService interface {
	CreateTask(ctx context.Context, taskID string, status string) error
	GetTask(ctx context.Context, taskID string) (models.Task, error)
	ChangeStatusTask(ctx context.Context, taskID string, status string, comment string) error
}

func New(
	log *slog.Logger,
	schemaProvider SchemaProvider,
	taskService TaskService,
	tableProvider TableProvider,
	DWHDbName string,
	OLTPDbName string,

) *AnalyticsDataCenterService {
	return &AnalyticsDataCenterService{
		log:            log,
		SchemaProvider: schemaProvider,
		TaskService:    taskService,
		TableProvider:  tableProvider,
		DWHDbName:      DWHDbName,
		OLTPDbName:     OLTPDbName,
	}
}

func (a *AnalyticsDataCenterService) StartETLProcess(ctx context.Context, idView int64) (taskID string, err error) {
	const op = "analytics.StartETLProcess"
	var queriesInit models.Queries
	log := a.log.With(
		slog.String("op", op),
		slog.Int64("idSchema", idView),
	)
	log.Info("ETL start")
	taskID = uuid.NewString()
	a.TaskService.CreateTask(ctx, taskID, Progress)
	viewSchema, err := a.SchemaProvider.GetView(ctx, idView)
	if err != nil {
		if errors.Is(err, storage.ErrSchemaNotFound) {
			a.log.Warn("view not found", slog.String("error", err.Error()))
			a.TaskService.ChangeStatusTask(ctx, taskID, Error, ErrorCreateTemplateTable)
			return "", fmt.Errorf("%s:%s", op, ErrInvalidSchemID)

		}
		a.TaskService.ChangeStatusTask(ctx, taskID, Error, ErrorCreateTemplateTable)
		a.log.Warn("view not found", slog.String("error", err.Error()))
		return "", fmt.Errorf("%s:%s", op, err)
	}
	if a.OLTPDbName == DbPostgres {
		queries, duplicates, err := sqlgenerator.GenerateQueryCreateTempTablePostgres(&viewSchema, log)
		if err != nil {
			log.Error("не удалось сгенерировать запросы генератором SQL", slog.String("error", err.Error()))
			a.TaskService.ChangeStatusTask(ctx, taskID, Error, ErrorCreateTemplateTable)
		}

		queriesInit = queries

		if len(duplicates) > 0 {
			// TO DO записать в комментарий к задачи, что были найдены дубликаты.

		}

	}
	if a.DWHDbName == DbPostgres {
		err = a.createTempTables(ctx, queriesInit)
		if err != nil {
			log.Error("не удалось создать временные таблицы", slog.String("error", err.Error()))
			a.TaskService.ChangeStatusTask(ctx, taskID, Error, ErrorCreateTemplateTable)
		}
	}

	return taskID, nil

}

func (a *AnalyticsDataCenterService) createTempTables(ctx context.Context, quries models.Queries) error {
	const op = "analytics.createTempTables"
	var errorCreate error
	log := a.log.With(
		slog.String("op", op),
	)

	for _, query := range quries.Queries {
		err := a.TableProvider.CreateTempTablePostgres(ctx, query.Query)
		if err != nil {
			errorCreate = err
			log.Error("не удалось создать временные таблицы", slog.String("error", err.Error()))
			break

		}

	}
	if errorCreate != nil {
		for _, tableQuery := range quries.Queries {
			err := a.TableProvider.DeleteTempTablePostgres(ctx, tableQuery.TableName)
			if err != nil {
				//TO DO сделать worker , который раз в какое-то время запускается и чистит темп таблицы на такие случаи.
				log.Error("не удалось удалить временную таблицу",
					slog.String("table", tableQuery.TableName),
					slog.String("error", err.Error()),
				)
			}
		}
		return errorCreate

	}

	return nil

}
