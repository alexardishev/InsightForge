package main

import (
	"analyticDataCenter/analytics-data-center/internal/app"
	"analyticDataCenter/analytics-data-center/internal/config"
	"fmt"
	"log"
	"log/slog"
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

	fmt.Println(cfg)
	logger := setupLogger(cfg.Env)
	logger.Info("Starting Analytics server")

	application := app.New(logger, cfg.GRPC.Port, cfg.StoragePath, cfg.OLTPStoragePath, cfg.DWHStoragePath, cfg.OLTPDataBase, cfg.DWHDataBase, cfg.TokenTTL, cfg.OLTPstorages)

	go application.GRPCSrv.Run()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT, &customSignal)

	sign := <-stop
	logger.Info("stopping application", slog.String("signal", sign.String()))

	application.GRPCSrv.Stop()

	logger.Info("app stoped")
}

func setupLogger(env string) *slog.Logger {
	var logger *slog.Logger

	switch env {
	case "development":
		logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	case "production":
		logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	case "local":
		logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	case "test":
	default:
		log.Fatalf("Invalid environment: %s", env)
	}
	return logger

}
