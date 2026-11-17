package storage

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"analyticDataCenter/analytics-data-center/internal/config"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

func TestOLTPFactoryDBReusesSourcePool(t *testing.T) {
	// sql.Open не подключается к базе, поэтому адрес может быть недоступен.
	f := NewOLTPFactory(slog.New(slog.NewTextHandler(io.Discard, nil)), []config.OLTPstorage{
		{Name: "db1", Path: "postgresql://u:p@127.0.0.1:1/db?sslmode=disable"},
	})
	defer f.CloseAll()

	db, err := f.DB(context.Background(), "db1")
	require.NoError(t, err)
	require.NotNil(t, db)
	again, err := f.DB(context.Background(), "db1")
	require.NoError(t, err)
	require.Same(t, db, again, "one connection pool per source")

	_, err = f.DB(context.Background(), "unknown")
	require.Error(t, err)
}
