package app

import (
	grpcapp "analyticDataCenter/analytics-data-center/internal/app/grpc"
	"analyticDataCenter/analytics-data-center/internal/config"
	"analyticDataCenter/analytics-data-center/internal/kafkaengine"
	serviceanalytics "analyticDataCenter/analytics-data-center/internal/services/analytics"
	tasksserivce "analyticDataCenter/analytics-data-center/internal/services/tasks"
	"analyticDataCenter/analytics-data-center/internal/storage"
	"analyticDataCenter/analytics-data-center/internal/storage/postgres"
	postgresdwh "analyticDataCenter/analytics-data-center/internal/storage/postgresDWH"
	"log/slog"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	_ "github.com/lib/pq"
)

const (
	DbPostgres   = "postgres"
	DbClickhouse = "clickhouse"
)

type App struct {
	GRPCSrv     *grpcapp.App
	OLTPFactory *storage.InstanceOLTPFactory
	Kafka       *kafka.Consumer
}

func New(log *slog.Logger, grpcPort int,
	storagePath string, connectionStringOLTP string, connectionStringDWH string, OLTPName string, DWHName string,
	tokenTTL time.Duration, factoryOLTP []config.OLTPstorage, BootstrapServers string, GroupId string, AutoOffsetReset string, EnableAutoCommit string, SessionTimeoutMs string, ClientId string) *App {
	// TO DO переделать на cfg
	statusEnum := []string{"In progress", "Execution error", "Completed"}
	// var storageOLTP storage.OLTPDB
	var storageDWH storage.DWHDB
	storageSys, err := postgres.New(storagePath, log)
	if err != nil {
		panic("Не удалось создать Storage SYS")
	}

	kafkaConsumer, err := kafkaengine.NewKafkaConsumer(BootstrapServers, GroupId, AutoOffsetReset, EnableAutoCommit, SessionTimeoutMs, ClientId, log)
	if err != nil {
		panic("Не удалось подключиться к Kafka")
	}

	if DWHName == DbPostgres {
		var storageDWHPostgres *postgresdwh.PostgresDWH
		storageDWHPostgres, err = postgresdwh.New(connectionStringDWH, log)
		if err != nil {
			panic("Не удалось создать Storage DWH")
		}
		storageDWH = storageDWHPostgres
	}

	oltpFactory := storage.NewOLTPFactory(log, factoryOLTP)
	if DWHName == DbClickhouse {
		//TO DO дописать когда появится адаптер для Clickhouse
	}

	storage, err := storage.New(storageSys, log, storageDWH)
	if err != nil {
		panic("Не удалось создать Storage")
	}
	tasksserivce := tasksserivce.New(log, storageSys, statusEnum)
	analyticsService := serviceanalytics.New(log, storage.DbSys, tasksserivce, storageDWH, oltpFactory, DWHName, OLTPName)
	grpcServer := grpcapp.New(log, grpcPort, analyticsService)
	return &App{
		GRPCSrv:     grpcServer,
		OLTPFactory: oltpFactory,
		Kafka:       kafkaConsumer,
	}
}
