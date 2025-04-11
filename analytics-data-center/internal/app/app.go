package app

import (
	grpcapp "analyticDataCenter/analytics-data-center/internal/app/grpc"
	serviceanalytics "analyticDataCenter/analytics-data-center/internal/services/analytics"
	tasksserivce "analyticDataCenter/analytics-data-center/internal/services/tasks"
	"analyticDataCenter/analytics-data-center/internal/storage/postgres"
	"log/slog"
	"time"

	_ "github.com/lib/pq"
)

type App struct {
	GRPCSrv *grpcapp.App
}

func New(log *slog.Logger, grpcPort int,
	storagePath string, connectionStringOLTP string, connectionStringDWH string, OLTPName string, DWHName string,
	tokenTTL time.Duration) *App {
	// TO DO переделать на cfg
	statusEnum := []string{"In progress", "Execution error", "Completed"}

	storage, err := postgres.New(storagePath, log, connectionStringOLTP, connectionStringDWH, OLTPName, DWHName)
	if err != nil {
		panic("Не удалось создать Storage")
	}
	tasksserivce := tasksserivce.New(log, storage, statusEnum)
	analyticsService := serviceanalytics.New(log, storage, tasksserivce, storage, DWHName, OLTPName)
	grpcServer := grpcapp.New(log, grpcPort, analyticsService)
	return &App{
		GRPCSrv: grpcServer,
	}
}
