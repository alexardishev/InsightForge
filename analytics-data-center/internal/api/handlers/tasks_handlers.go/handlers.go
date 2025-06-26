package taskshandlers

import (
	"analyticDataCenter/analytics-data-center/internal/domain/models"
	"analyticDataCenter/analytics-data-center/internal/lib/validate"
	serviceanalytics "analyticDataCenter/analytics-data-center/internal/services/analytics"
	"encoding/json"
	"log/slog"
	"net/http"

	loggerpkg "analyticDataCenter/analytics-data-center/internal/logger"
)

type TaskHandlers struct {
	log              *loggerpkg.Logger
	serviceAnalytics *serviceanalytics.AnalyticsDataCenterService
}

func NewTaskHandlers(log *loggerpkg.Logger, serviceAnalytics *serviceanalytics.AnalyticsDataCenterService,
) *TaskHandlers {
	return &TaskHandlers{
		log:              log,
		serviceAnalytics: serviceAnalytics,
	}
}

func (t *TaskHandlers) GetTasks(w http.ResponseWriter, r *http.Request) {
	const op = "DBHandlers.GetConnectionsString"
	t.log.With(
		slog.String("op", op),
	)
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")
	var taskFilters models.TaskFilter
	if err := json.NewDecoder(r.Body).Decode(&taskFilters); err != nil {
		t.log.Error("failed to decode request", slog.String("error", err.Error()))
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	t.log.Info("фильтр", slog.Any("filter", taskFilters))
	_, err := validate.Validate(taskFilters)
	if err != nil {
		t.log.Error("failed validate", slog.String("error", err.Error()))
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	tasks, err := t.serviceAnalytics.GetTasks(ctx, taskFilters)
	if err != nil {
		t.log.Error("ошибка сервиса аналитики", slog.String("error", err.Error()))
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(tasks); err != nil {
		t.log.Error("failed to encode response", slog.String("error", err.Error()))
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
}
