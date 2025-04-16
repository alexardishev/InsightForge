package storage

import (
	"analyticDataCenter/analytics-data-center/internal/config"
	postgresoltp "analyticDataCenter/analytics-data-center/internal/storage/postgresOLTP"
	"context"
	"fmt"
	"log/slog"
)

type OLTPFactory interface {
	GetOLTPStorage(ctx context.Context, sourceName string) (OLTPDB, error)
}

type PostgresOLTPFactory struct {
	logger      *slog.Logger
	connections map[string]string // sourceName → connection string
}

func NewOLTPFactory(logger *slog.Logger, connConfigs []config.OLTPstorage) *PostgresOLTPFactory {
	connMap := make(map[string]string)
	for _, c := range connConfigs {
		connMap[c.Name] = c.Path
	}
	return &PostgresOLTPFactory{
		logger:      logger,
		connections: connMap,
	}
}

func (f *PostgresOLTPFactory) GetOLTPStorage(ctx context.Context, sourceName string) (OLTPDB, error) {
	connStr, ok := f.connections[sourceName]
	if !ok {
		return nil, fmt.Errorf("connection string for %s not found", sourceName)
	}
	return postgresoltp.New(connStr, f.logger)
}
