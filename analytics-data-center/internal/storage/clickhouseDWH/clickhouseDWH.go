package clickhousedwh

import (
	"database/sql"
	"log/slog"

	_ "github.com/ClickHouse/clickhouse-go/v2"
)

type ClickHouseDB struct {
	Db  *sql.DB
	Log *slog.Logger
}

func New(connectionString string, log *slog.Logger) (*ClickHouseDB, error) {
	const op = "storage.ClickHouseDB.New"
	log = log.With(
		slog.String("op", op),
	)

	db, err := sql.Open("clickhouse", connectionString)
	if err != nil {
		log.Error("получена ошибка при коннекте к БД", slog.String("error", err.Error()))
		return nil, err
	}
	return &ClickHouseDB{Db: db, Log: log}, nil
}
