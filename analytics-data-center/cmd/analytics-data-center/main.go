package main

import (
	"analyticDataCenter/analytics-data-center/internal/app"
	"analyticDataCenter/analytics-data-center/internal/config"
	"analyticDataCenter/analytics-data-center/internal/logger"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// cdcShutdownTimeout — сколько ждать завершения текущей CDC-обработки при остановке.
const cdcShutdownTimeout = 30 * time.Second

type CustomSignal string

func (c *CustomSignal) Signal() {
}

func (c *CustomSignal) String() string {
	return ""
}

func main() {

	customSignal := CustomSignal("MyCustomSignal")

	cfg := config.MustLoad()

	logg := logger.New(cfg.Env, cfg.LogLang)
	logg.InfoMsg(logger.MsgAnalyticsServerStart)

	application := app.New(logg, cfg.GRPC.Port, cfg.StoragePath, cfg.OLTPStoragePath, cfg.DWHStoragePath, cfg.OLTPDataBase, cfg.DWHDataBase, cfg.DWHStoragePath, cfg.RenameHeuristic, cfg.TokenTTL, cfg.OLTPstorages, cfg.Kafka.BootstrapServers, cfg.Kafka.GroupId, cfg.Kafka.AutoOffsetReset, cfg.Kafka.EnableAutoCommit, cfg.Kafka.SessionTimeoutMs, cfg.Kafka.ClientId, cfg.Kafka.DLQRetention, cfg.Kafka.DebeziumTopicRetention, cfg.KafkaConnect, cfg.TopicSubscriptionInterval, cfg.SMTP.Host, cfg.SMTP.Port, cfg.SMTP.UserName, cfg.SMTP.Password, cfg.SMTP.AdminEmail, cfg.SMTP.FromEmail)

	go application.GRPCSrv.Run()
	go func() {
		logg.InfoMsg(logger.MsgHTTPServerStarted)
		err := http.ListenAndServe(":8888", application.Router)
		if err != nil {
			panic("❌ Failed to start server:")
		}
	}()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT, &customSignal)

	var cdcErr error
	select {
	case sign := <-stop:
		logg.InfoMsg(logger.MsgStoppingApplication, slog.String("signal", sign.String()))
		application.StopCDC()
		// Ждём выхода listener-а: закрывать consumer, пока Run внутри Poll, нельзя.
		select {
		case cdcErr = <-application.CDCErrors:
			if errors.Is(cdcErr, context.Canceled) {
				cdcErr = nil
			}
		case <-time.After(cdcShutdownTimeout):
			// Listener ещё работает, закрывать consumer небезопасно — выходим сразу.
			// Offset текущего сообщения не закоммичен, после рестарта оно придёт снова.
			logg.ErrorMsg(logger.MsgCDCShutdownTimeout, slog.Duration("timeout", cdcShutdownTimeout))
			os.Exit(1)
		}
	case cdcErr = <-application.CDCErrors:
		application.StopCDC()
	}
	if cdcErr != nil {
		logg.ErrorMsg(logger.MsgCDCStopped, slog.String("error", cdcErr.Error()))
		// Фатальная ошибка CDC завершает процесс без полного graceful shutdown.
		// Listener уже остановлен, поэтому Kafka закрываем: consumer сразу выходит
		// из группы, и партиции переназначаются без ожидания session.timeout.
		stopKafka(logg, application)
		os.Exit(1)
	}

	application.GRPCSrv.Stop()
	stopKafka(logg, application)

	if err := application.OLTPFactory.CloseAll(); err != nil {
		logg.Error("OLTPFactory close failed", slog.String("error", err.Error()))
	}

	logg.InfoMsg(logger.MsgApplicationStopped)
}

// stopKafka останавливает подписку на новые топики, дожидается отправки DLQ
// и закрывает Kafka (admin-клиент, затем consumer). Вызывать только после остановки cdc.Listener.
func stopKafka(log *logger.Logger, application *app.App) {
	if application.TopicCron != nil {
		application.TopicCron.Stop()
	}
	application.DLQ.Close()
	if err := application.KafkaEngine.Close(); err != nil {
		log.ErrorMsg(logger.MsgKafkaCloseFailed, slog.String("error", err.Error()))
	}
}
