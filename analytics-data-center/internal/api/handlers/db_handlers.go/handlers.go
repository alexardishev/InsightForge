package dbhandlers

import (
	"analyticDataCenter/analytics-data-center/internal/domain/models"
	"analyticDataCenter/analytics-data-center/internal/lib/validate"
	serviceanalytics "analyticDataCenter/analytics-data-center/internal/services/analytics"
	"encoding/json"
	"log/slog"
	"net/http"

	loggerpkg "analyticDataCenter/analytics-data-center/internal/logger"
)

type DBHandlers struct {
	log              *loggerpkg.Logger
	serviceAnalytics *serviceanalytics.AnalyticsDataCenterService
}

func NewDBHandler(log *loggerpkg.Logger, serviceAnalytics *serviceanalytics.AnalyticsDataCenterService) *DBHandlers {
	return &DBHandlers{
		log:              log,
		serviceAnalytics: serviceAnalytics,
	}

}

func (d *DBHandlers) GetConnectionsStrings(w http.ResponseWriter, r *http.Request) {
	const op = "DBHandlers.GetConnectionsString"
	d.log.With(
		slog.String("op", op),
	)
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	OLTPDataBaseStrings := d.serviceAnalytics.OLTPFactory.GetOLTPStrings(ctx)
	if err := json.NewEncoder(w).Encode(OLTPDataBaseStrings); err != nil {
		d.log.Error("failed to encode response", slog.String("error", err.Error()))
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
}

func (d *DBHandlers) GetDBInformations(w http.ResponseWriter, r *http.Request) {
	const op = "DBHandlers.GetDBInformations"
	d.log.With(
		slog.String("op", op),
	)
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	var connectionsStrings models.ConnectionStrings
	if err := json.NewDecoder(r.Body).Decode(&connectionsStrings); err != nil {
		d.log.Error("failed to decode request", slog.String("error", err.Error()))
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	_, err := validate.Validate(connectionsStrings)
	if err != nil {
		d.log.Error("failed validate", slog.String("error", err.Error()))
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	dbNames, err := d.serviceAnalytics.GetDBInformations(ctx, connectionsStrings)
	if err != nil {
		d.log.Error("ошибка сервиса аналитики", slog.String("error", err.Error()))
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(dbNames); err != nil {
		d.log.Error("failed to encode response", slog.String("error", err.Error()))
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

}

func (d *DBHandlers) UploadSchema(w http.ResponseWriter, r *http.Request) {
	const op = "DBHandlers.UploadSchem"
	d.log.With(
		slog.String("op", op),
	)
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")
	var schemaView models.View
	if err := json.NewDecoder(r.Body).Decode(&schemaView); err != nil {
		d.log.Error("failed to decode request", slog.String("error", err.Error()))
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	_, err := validate.Validate(schemaView)
	if err != nil {
		d.log.Error("failed validate", slog.String("error", err.Error()))
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	id, err := d.serviceAnalytics.UploadSchema(ctx, schemaView)
	if err != nil {
		d.log.Error("ошибка сервиса аналитики", slog.String("error", err.Error()))
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(id); err != nil {
		d.log.Error("failed to encode response", slog.String("error", err.Error()))
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

}
