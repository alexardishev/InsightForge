package app

import (
	grpcapp "analyticDataCenter/analytics-data-center/internal/app/grpc"
	serviceanalytics "analyticDataCenter/analytics-data-center/internal/services/analytics"
	tasksserivce "analyticDataCenter/analytics-data-center/internal/services/tasks"
	"analyticDataCenter/analytics-data-center/internal/storage"
	"analyticDataCenter/analytics-data-center/internal/storage/postgres"
	postgresdwh "analyticDataCenter/analytics-data-center/internal/storage/postgresDWH"
	"log/slog"
	"time"

	_ "github.com/lib/pq"
)

const (
	DbPostgres   = "postgres"
	DbClickhouse = "clickhouse"
)

type App struct {
	GRPCSrv *grpcapp.App
}

func New(log *slog.Logger, grpcPort int,
	storagePath string, connectionStringOLTP string, connectionStringDWH string, OLTPName string, DWHName string,
	tokenTTL time.Duration) *App {
	// TO DO переделать на cfg
	statusEnum := []string{"In progress", "Execution error", "Completed"}
	// var storageOLTP *sql.DB
	var storageDWH storage.DWHDB
	storageSys, err := postgres.New(storagePath, log)
	if err != nil {
		panic("Не удалось создать Storage SYS")
	}

	//Сейчас нет OLTP,

	// if OLTPName == DbPostgres {
	// 	var storageOLTPPostgres *postgresoltp.PostgresOLTP
	// 	storageOLTPPostgres, err = postgresoltp.New(connectionStringOLTP, log)
	// 	if err != nil {
	// 		panic("Не удалось создать Storage OLTP")
	// 	}
	// 	storageOLTP = storageOLTPPostgres.Db
	// }

	if DWHName == DbPostgres {
		var storageDWHPostgres *postgresdwh.PostgresDWH
		storageDWHPostgres, err = postgresdwh.New(connectionStringDWH, log)
		if err != nil {
			panic("Не удалось создать Storage DWH")
		}
		storageDWH = storageDWHPostgres
	}

	storage, err := storage.New(storageSys, log, storageDWH)
	if err != nil {
		panic("Не удалось создать Storage")
	}
	tasksserivce := tasksserivce.New(log, storageSys, statusEnum)
	analyticsService := serviceanalytics.New(log, storage.DbSys, tasksserivce, storageDWH, DWHName, OLTPName)
	grpcServer := grpcapp.New(log, grpcPort, analyticsService)
	return &App{
		GRPCSrv: grpcServer,
	}
}
