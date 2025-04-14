package postgresoltp

import (
	"database/sql"
	"log/slog"
)

type PostgresOLTP struct {
	Db  *sql.DB
	Log *slog.Logger
}

func New(connectionString string, log *slog.Logger) (*PostgresOLTP, error) {
	const op = "storage.Postgres.New"
	log = log.With(
		slog.String("op", op),
	)

	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		log.Error("получена ошибка при коннекте к БД", slog.String("error", err.Error()))
		return nil, err
	}

	return &PostgresOLTP{
		Db:  db,
		Log: log,
	}, nil
}
