package main

import (
	"errors"
	"flag"

	loggerpkg "analyticDataCenter/analytics-data-center/internal/logger"

	"github.com/golang-migrate/migrate"
	_ "github.com/golang-migrate/migrate/database/postgres"
	_ "github.com/golang-migrate/migrate/database/sqlite3"
	_ "github.com/golang-migrate/migrate/source/file"
)

func main() {
	var storagePath, migrationsPath, driverSql string
	flag.StringVar(&storagePath, "storage-path", "", "path to storage")
	flag.StringVar(&migrationsPath, "migrations-path", "", "path to migrations")
	flag.StringVar(&driverSql, "driver", "postgres", "SQL driver") // flag.StringVar(&migrationsTable, "migrations-table", "migrations", "name of migrations tables")
	flag.Parse()

	if storagePath == "" {
		panic("strpath is req")
	}

	if migrationsPath == "" {
		panic("migration path is req")
	}

	logger := loggerpkg.New("development", "ru")

	m, err := migrate.New("file://"+migrationsPath, storagePath)
	if err != nil {
		panic(err)
	}

	err = m.Up()
	if err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			logger.InfoMsg(loggerpkg.MsgNoChanges)
			return
		}
		panic(err)
	}

	logger.InfoMsg(loggerpkg.MsgMigrationsCompleted)

}
