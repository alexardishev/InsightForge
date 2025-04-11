package postgres

import (
	"database/sql"
	"log/slog"

	_ "github.com/golang-migrate/migrate/database/postgres"
	_ "github.com/golang-migrate/migrate/source/file"
	_ "github.com/lib/pq"
)

type Storage struct {
	db     *sql.DB
	log    *slog.Logger
	dbOLTP *sql.DB
	dbDWH  *sql.DB
}

func New(connectionString string, log *slog.Logger, connectionStringOLTP string, connectionStringDWH string, OLTPName string, DWHName string) (*Storage, error) {
	const op = "storage.Postgres.New"
	log = log.With(
		slog.String("op", op),
	)

	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		log.Error("получена ошибка при коннекте к БД", slog.String("error", err.Error()))
		return nil, err
	}

	dbOLTP, err := sql.Open(OLTPName, connectionStringOLTP)
	if err != nil {
		log.Error("получена ошибка при коннекте к БД OLTP", slog.String("error", err.Error()))
		return nil, err
	}

	dbDWH, err := sql.Open(DWHName, connectionStringDWH)
	if err != nil {
		log.Error("получена ошибка при коннекте к БД DWH", slog.String("error", err.Error()))
		return nil, err
	}

	return &Storage{
		db:     db,
		dbOLTP: dbOLTP,
		dbDWH:  dbDWH,
		log:    log,
	}, nil
}
