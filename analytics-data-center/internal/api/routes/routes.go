package routes

import (
	"analyticDataCenter/analytics-data-center/internal/api/handlers"
	dbhandlers "analyticDataCenter/analytics-data-center/internal/api/handlers/db_handlers.go"
	"analyticDataCenter/analytics-data-center/internal/api/middleware"
	serviceanalytics "analyticDataCenter/analytics-data-center/internal/services/analytics"
	"net/http"

	loggerpkg "analyticDataCenter/analytics-data-center/internal/logger"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
)

func NewRouter(logger *loggerpkg.Logger, serviceAnalytics *serviceanalytics.AnalyticsDataCenterService) http.Handler {
	r := chi.NewRouter()
	dbhandlers := dbhandlers.NewDBHandler(logger, serviceAnalytics)
	handlers := handlers.NewHandlers(logger, dbhandlers)

	logMiddleware := middleware.NewLogger(logger)
	r.Use(logMiddleware.Middleware)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	r.Route("/api", func(r chi.Router) {
		r.Get("/get-connections", handlers.GetConnectionsStrings)
		r.Post("/get-db", handlers.GetDBInformations)
		r.Post("/upload-schem", handlers.UploadSchema)
	})

	return r
}
