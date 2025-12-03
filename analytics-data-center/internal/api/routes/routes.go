package routes

import (
	"analyticDataCenter/analytics-data-center/internal/api/handlers"
	dbhandlers "analyticDataCenter/analytics-data-center/internal/api/handlers/db_handlers.go"
	taskshandlers "analyticDataCenter/analytics-data-center/internal/api/handlers/tasks_handlers.go"
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
	taskshandlers := taskshandlers.NewTaskHandlers(logger, serviceAnalytics)

	handlers := handlers.NewHandlers(logger, dbhandlers, taskshandlers)

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
		r.Post("/get-tasks", handlers.GetTasks)
		r.Get("/column-rename-suggestions", handlers.GetColumnRenameSuggestions)
		r.Post("/column-rename-suggestions/{id}/accept", handlers.AcceptColumnRenameSuggestion)
		r.Post("/column-rename-suggestions/{id}/reject", handlers.RejectColumnRenameSuggestion)
		r.Get("/column-mismatch-groups", handlers.GetColumnMismatchGroups)
		r.Get("/column-mismatch-groups/{id}", handlers.GetColumnMismatchGroup)
		r.Post("/column-mismatch-groups/{id}/apply", handlers.ApplyColumnMismatchGroup)
	})

	return r
}
