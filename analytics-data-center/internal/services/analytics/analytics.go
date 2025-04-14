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
)

type AnalyticsDataCenterService struct {
	log            *slog.Logger
	SchemaProvider storage.SysDB
	TaskService    TaskService
	TableProvider  storage.DWHDB
	DWHDbName      string
	OLTPDbName     string
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
	tableProvider storage.DWHDB,
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
			log.Warn("duplicates", slog.String("duplicates", strings.Join(duplicates, ",")))

		}

	}
	err = a.createTempTables(ctx, queriesInit)
	if err != nil {
		log.Error("не удалось создать временные таблицы", slog.String("error", err.Error()))
		a.TaskService.ChangeStatusTask(ctx, taskID, Error, ErrorCreateTemplateTable)
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
		err := a.TableProvider.CreateTempTable(ctx, query.Query, query.TableName)
		if err != nil {
			errorCreate = err
			log.Error("не удалось создать временные таблицы", slog.String("error", err.Error()))
			break

		}

	}
	if errorCreate != nil {
		for _, tableQuery := range quries.Queries {
			err := a.TableProvider.DeleteTempTable(ctx, tableQuery.TableName)
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

// func (a *AnalyticsDataCenterService) getCountInsertData(ctx context.Context, viewSchema models.View) (int64, error) {
// 	const op = "analytics.getCountInsertData"
// 	log := a.log.With(
// 		slog.String("op", op),
// 	)
// 	if a.OLTPDbName == DbPostgres {
// 		a.getCountInsertData

// 	}

// }
