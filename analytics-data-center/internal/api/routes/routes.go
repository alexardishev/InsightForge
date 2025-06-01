package routes

import (
	"analyticDataCenter/analytics-data-center/internal/api/handlers"
	dbhandlers "analyticDataCenter/analytics-data-center/internal/api/handlers/db_handlers.go"
	serviceanalytics "analyticDataCenter/analytics-data-center/internal/services/analytics"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func NewRouter(logger *slog.Logger, serviceAnalytics *serviceanalytics.AnalyticsDataCenterService) http.Handler {
	r := chi.NewRouter()
	dbhandlers := dbhandlers.NewDBHandler(logger, serviceAnalytics)
	handlers := handlers.NewHandlers(logger, dbhandlers)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	r.Route("/api", func(r chi.Router) {
		r.Get("/get-connections", handlers.GetConnectionsStrings)
		r.Post("/get-db", handlers.GetDB)
	})

	return r
}
