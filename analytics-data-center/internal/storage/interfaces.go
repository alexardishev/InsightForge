// internal/storage/interfaces.go

package storage

import (
	"analyticDataCenter/analytics-data-center/internal/domain/models"
	"context"
)

// // DB — общий интерфейс для низкоуровневых операций, если нужно
//
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
type SchemaProvider interface {
	GetView(ctx context.Context, idView int64) (models.View, error)
}

type TaskProvider interface {
	CreateTask(ctx context.Context, taskID string, status string) error
	GetTask(ctx context.Context, taskID string) (models.Task, error)
	ChangeStatusTask(ctx context.Context, taskID string, newStatus string, comment string) error
}

type TableProvider interface {
	CreateTempTablePostgres(ctx context.Context, query string, tempTableName string) error
	DeleteTempTablePostgres(ctx context.Context, tableName string) error
}
