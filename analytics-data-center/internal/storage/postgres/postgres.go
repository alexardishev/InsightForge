package postgres

import (
	"database/sql"
	"log/slog"

	_ "github.com/golang-migrate/migrate/database/postgres"
	_ "github.com/golang-migrate/migrate/source/file"
	_ "github.com/lib/pq"
)

type PostgresSys struct {
	Db  *sql.DB
	Log *slog.Logger
}

func New(connectionString string, log *slog.Logger) (*PostgresSys, error) {
	const op = "storage.Postgres.New"
	log = log.With(
		slog.String("op", op),
	)

	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		log.Error("получена ошибка при коннекте к БД", slog.String("error", err.Error()))
		return nil, err
	}

	return &PostgresSys{
		Db:  db,
		Log: log,
	}, nil
}
