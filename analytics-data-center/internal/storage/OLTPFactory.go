package storage

import (
	"analyticDataCenter/analytics-data-center/internal/config"
	postgresoltp "analyticDataCenter/analytics-data-center/internal/storage/postgresOLTP"
	"context"
	"fmt"
	"log/slog"
	"sync"
)

type OLTPFactory interface {
	GetOLTPStorage(ctx context.Context, sourceName string) (OLTPDB, error)
	CloseAll() error // чтобы можно было корректно закрыть соединения при завершении программы
}

type InstanceOLTPFactory struct {
	logger           *slog.Logger
	connections      map[string]string
	connectionsKafka map[string]string                     // sourceName → connection string
	pool             map[string]*postgresoltp.PostgresOLTP // sourceName → готовое подключение
	mu               sync.Mutex
}

func NewOLTPFactory(logger *slog.Logger, connConfigs []config.OLTPstorage) *InstanceOLTPFactory {
	connMap := make(map[string]string)
	connMapKafka := make(map[string]string)

	for _, c := range connConfigs {
		connMap[c.Name] = c.Path
		connMapKafka[c.Name] = c.PathKafka
		fmt.Println(c.PathKafka)
	}
	return &InstanceOLTPFactory{
		logger:           logger,
		connections:      connMap,
		connectionsKafka: connMapKafka,
		pool:             make(map[string]*postgresoltp.PostgresOLTP),
	}
}

func (f *InstanceOLTPFactory) GetOLTPStorage(ctx context.Context, sourceName string) (OLTPDB, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if conn, ok := f.pool[sourceName]; ok {
		return conn, nil
	}

	// Создаём, если нет
	connStr, ok := f.connections[sourceName]
	if !ok {
		return nil, fmt.Errorf("connection string for %s not found", sourceName)
	}

	storage, err := postgresoltp.New(connStr, f.logger)
	if err != nil {
		return nil, err
	}

	storage.Db.SetMaxOpenConns(10)
	storage.Db.SetMaxIdleConns(5)

	f.pool[sourceName] = storage
	return storage, nil
}

// Закрыть все соединения при завершении работы
func (f *InstanceOLTPFactory) CloseAll() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for name, s := range f.pool {
		f.logger.Info("Закрытие соединения", slog.String("source", name))
		if err := s.Db.Close(); err != nil {
			return err
		}
	}
	return nil
}
