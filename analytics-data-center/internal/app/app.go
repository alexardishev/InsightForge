package app

import (
	"analyticDataCenter/analytics-data-center/internal/api/routes"
	grpcapp "analyticDataCenter/analytics-data-center/internal/app/grpc"
	"analyticDataCenter/analytics-data-center/internal/config"
	"analyticDataCenter/analytics-data-center/internal/kafkaengine"
	"analyticDataCenter/analytics-data-center/internal/notifications"
	serviceanalytics "analyticDataCenter/analytics-data-center/internal/services/analytics"
	"analyticDataCenter/analytics-data-center/internal/services/cdc"
	"analyticDataCenter/analytics-data-center/internal/services/debezium"
	smtpsender "analyticDataCenter/analytics-data-center/internal/services/smtrsender"
	tasksserivce "analyticDataCenter/analytics-data-center/internal/services/tasks"
	"analyticDataCenter/analytics-data-center/internal/services/topicsubscription"
	"analyticDataCenter/analytics-data-center/internal/storage"
	clickhousedwh "analyticDataCenter/analytics-data-center/internal/storage/clickhouseDWH"
	"analyticDataCenter/analytics-data-center/internal/storage/postgres"
	postgresdwh "analyticDataCenter/analytics-data-center/internal/storage/postgresDWH"
	"context"
	"log/slog"

	loggerpkg "analyticDataCenter/analytics-data-center/internal/logger"
	"net/http"
	"time"

	_ "github.com/lib/pq"
)

const (
	DbPostgres   = "postgres"
	DbClickhouse = "clickhouse"
)

type App struct {
	GRPCSrv       *grpcapp.App
	OLTPFactory   *storage.InstanceOLTPFactory
	KafkaEngine   *kafkaengine.Engine
	DLQ           *kafkaengine.DLQProducer
	Router        http.Handler
	TopicCron     *topicsubscription.Cron
	Notifications *notifications.Worker
	CDCErrors     <-chan error
	StopCDC       context.CancelFunc
}

func New(log *loggerpkg.Logger, grpcPort int,
	storagePath string, connectionStringOLTP string, connectionStringDWH string, OLTPName string, DWHName string, DWHPath string,
	renameHeuristic bool,
	tokenTTL time.Duration, factoryOLTP []config.OLTPstorage, BootstrapServers string, GroupId string, AutoOffsetReset string, EnableAutoCommit string, SessionTimeoutMs string, ClientId string, dlqRetention time.Duration, topicRetention time.Duration, KafkaConnect string, topicSubscriptionInterval time.Duration, hostSMTP string, portSMTP int, userNameSMTP string, passwordSMTP string, adminEmailSMTP string, fromEmailSMTP string) *App {
	// TO DO переделать на cfg
	statusEnum := []string{"In progress", "Execution error", "Completed"}
	// var storageOLTP storage.OLTPDB
	var storageDWH storage.DWHDB
	storageSys, err := postgres.New(storagePath, log.Logger)
	if err != nil {
		panic("Не удалось создать Storage SYS")
	}
	smtp := smtpsender.NewSMTP(hostSMTP, portSMTP, userNameSMTP, passwordSMTP, adminEmailSMTP, fromEmailSMTP, log)
	if DWHName == DbPostgres {
		var storageDWHPostgres *postgresdwh.PostgresDWH
		storageDWHPostgres, err = postgresdwh.New(connectionStringDWH, log.Logger)
		if err != nil {
			panic("Не удалось создать Storage DWH")
		}
		storageDWH = storageDWHPostgres
	}

	oltpFactory := storage.NewOLTPFactory(log.Logger, factoryOLTP)
	if DWHName == DbClickhouse {
		var storageDWHClickHouse *clickhousedwh.ClickHouseDB
		storageDWHClickHouse, err = clickhousedwh.New(connectionStringDWH, log.Logger)
		if err != nil {
			panic("Не удалось создать Storage DWH")
		}
		storageDWH = storageDWHClickHouse
	}

	var registeredConnectors []config.OLTPstorage

	for _, conn := range factoryOLTP {
		err := debezium.RegisterPostgresConnector(KafkaConnect, conn.Name, conn.PathKafka, debezium.ConnectorOptions{TopicRetention: topicRetention})
		if err != nil {
			log.Error("Debezium не смог зарегистрировать коннектор", slog.String("name", conn.Name), slog.String("error", err.Error()))
			continue
		}
		registeredConnectors = append(registeredConnectors, conn)
	}

	debezium.WaitConnectorsReady(KafkaConnect, registeredConnectors, log)
	storage, err := storage.New(storageSys, log, storageDWH)
	if err != nil {
		panic("Не удалось создать Storage")
	}
	tasksserivce := tasksserivce.New(log, storageSys, statusEnum)
	analyticsService := serviceanalytics.New(log, storage.DbSys, tasksserivce, storageDWH, oltpFactory, DWHName, DWHPath, OLTPName, renameHeuristic, *smtp)
	notificationWorker := notifications.NewWorker(log)
	analyticsService.SetNotifier(notificationWorker)
	r := routes.NewRouter(log, analyticsService, notificationWorker)

	kafkaEngine, err := kafkaengine.NewEngine(BootstrapServers, GroupId, AutoOffsetReset, EnableAutoCommit, SessionTimeoutMs, ClientId, topicRetention, storage.DbSys, log)
	if err != nil {
		log.ErrorMsg(loggerpkg.MsgKafkaInitFailed, slog.String("error", err.Error()))
		panic("Не удалось подключиться к Kafka")
	}
	dlq, err := kafkaengine.NewDLQProducer(BootstrapServers, ClientId, dlqRetention, log)
	if err != nil {
		log.ErrorMsg(loggerpkg.MsgKafkaInitFailed, slog.String("error", err.Error()))
		panic("Не удалось создать продюсер DLQ")
	}
	analyticsService.SetTopicNotifier(kafkaEngine)
	topicCron := topicsubscription.NewCron(log, storage.DbSys, kafkaEngine, topicSubscriptionInterval)
	topicCron.Start()
	kafkaConsumer := kafkaEngine.Consumer()
	cdcListener := cdc.NewListener(kafkaConsumer, dlq, log, func(ctx context.Context, data []byte) error {
		return cdc.Dispatch(ctx, data, log, analyticsService)
	})
	// Регистрируем до Start: rebalance-callback срабатывает только внутри Poll listener-а.
	if topicRetention > 0 {
		// Предупреждать, когда отставание дошло до половины срока хранения топиков.
		cdcListener.LagWarnAge = topicRetention / 2
	}
	kafkaEngine.OnPartitionsRevoked(cdcListener.PartitionsRevoked)
	cdcCtx, stopCDC := context.WithCancel(context.Background())
	cdcErrors := cdcListener.Start(cdcCtx)
	grpcServer := grpcapp.New(log, grpcPort, analyticsService)
	return &App{
		GRPCSrv:       grpcServer,
		OLTPFactory:   oltpFactory,
		KafkaEngine:   kafkaEngine,
		DLQ:           dlq,
		Router:        r,
		TopicCron:     topicCron,
		Notifications: notificationWorker,
		CDCErrors:     cdcErrors,
		StopCDC:       stopCDC,
	}
}
