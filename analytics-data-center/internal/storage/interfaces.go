// internal/storage/interfaces.go

package storage

import (
	"analyticDataCenter/analytics-data-center/internal/domain/models"
	"context"
)

//	type DB interface {
//		ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
//		QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
//	}
type SysDB interface {
	TaskProvider
	SchemaProvider
}
type DWHDB interface {
	TableProvider
}

type OLTPDB interface {
	DataProviderOLTP
}
type SchemaProvider interface {
	GetView(ctx context.Context, idView int64) (models.View, error)
}

type TaskProvider interface {
	CreateTask(ctx context.Context, taskID string, status string) error
	GetTask(ctx context.Context, taskID string) (models.Task, error)
	ChangeStatusTask(ctx context.Context, taskID string, newStatus string, comment string) error
}

type TableProvider interface {
	CreateTempTable(ctx context.Context, query string, tempTableName string) error
	DeleteTempTable(ctx context.Context, tableName string) error
}

type DataProviderOLTP interface {
	GetCountInsertData(ctx context.Context, query string) (int64, error) // count of insert datas
}
