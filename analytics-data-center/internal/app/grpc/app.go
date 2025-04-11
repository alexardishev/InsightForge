package grpcapp

import (
	analyticsrpc "analyticDataCenter/analytics-data-center/internal/grpc/analytics-data-center"

	"fmt"
	"log/slog"
	"net"

	"google.golang.org/grpc"
)

type App struct {
	log        *slog.Logger
	grpcServer *grpc.Server
	port       int
}

func New(log *slog.Logger, port int, analyticsDataCenterService analyticsrpc.AnalyticsDataCenter) *App {

	gRPCServer := grpc.NewServer()
	analyticsrpc.RegisterServerAPI(gRPCServer, analyticsDataCenterService)

	return &App{
		log:        log,
		grpcServer: gRPCServer,
		port:       port,
	}
}

func (a *App) Run() error {
	const op = "grpcapp.Run"

	log := a.log.With(
		slog.String("op", op),
		slog.Int("port", a.port),
	)

	l, err := net.Listen("tcp", fmt.Sprintf(":%d", a.port))
	if err != nil {
		log.Error("Port not allowed", slog.Any("Error", err), slog.String("op", op))
		return err
	}

	log.Info("Server is running", slog.String("addr", l.Addr().String()))

	err = a.grpcServer.Serve(l)
	if err != nil {
		log.Error("Error", slog.Any("Error", err), slog.String("op", op))
		return err
	}

	return nil

}

func (a *App) Stop() {
	const op = "grpcapp.Stop"

	a.log.With(
		slog.String("op", op),
		slog.Int("port", a.port),
	)
	a.grpcServer.GracefulStop()
}

func (a *App) GetGRPCServer() *grpc.Server {
	return a.grpcServer
}
