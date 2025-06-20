package main

import (
	"analyticDataCenter/analytics-data-center/internal/app"
	"analyticDataCenter/analytics-data-center/internal/config"
	"analyticDataCenter/analytics-data-center/internal/logger"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

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

	application := app.New(logg, cfg.GRPC.Port, cfg.StoragePath, cfg.OLTPStoragePath, cfg.DWHStoragePath, cfg.OLTPDataBase, cfg.DWHDataBase, cfg.DWHStoragePath, cfg.TokenTTL, cfg.OLTPstorages, cfg.Kafka.BootstrapServers, cfg.Kafka.GroupId, cfg.Kafka.AutoOffsetReset, cfg.Kafka.EnableAutoCommit, cfg.Kafka.SessionTimeoutMs, cfg.Kafka.ClientId, cfg.KafkaConnect, cfg.SMTP.Host, cfg.SMTP.Port, cfg.SMTP.UserName, cfg.SMTP.Password, cfg.SMTP.AdminEmail, cfg.SMTP.FromEmail)

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

	sign := <-stop
	logg.InfoMsg(
		logger.MsgStoppingApplication,
		slog.String("signal", sign.String()),
	)

	application.GRPCSrv.Stop()
	if err := application.Kafka.Close(); err != nil {
		logg.Error("Kafka close failed", slog.String("error", err.Error()))
	}

	if err := application.OLTPFactory.CloseAll(); err != nil {
		logg.Error("OLTPFactory close failed", slog.String("error", err.Error()))
	}

	logg.InfoMsg(logger.MsgApplicationStopped)
}
