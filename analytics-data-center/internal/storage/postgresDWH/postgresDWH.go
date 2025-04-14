package postgresdwh

import (
	"database/sql"
	"log/slog"
)

type PostgresDWH struct {
	Db  *sql.DB
	Log *slog.Logger
}

func New(connectionString string, log *slog.Logger) (*PostgresDWH, error) {
	const op = "storage.Postgres.New"
	log = log.With(
		slog.String("op", op),
	)

	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		log.Error("получена ошибка при коннекте к БД", slog.String("error", err.Error()))
		return nil, err
	}

	return &PostgresDWH{
		Db:  db,
		Log: log,
	}, nil
}
