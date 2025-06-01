package dbhandlers

import (
	"analyticDataCenter/analytics-data-center/internal/domain/models"
	"analyticDataCenter/analytics-data-center/internal/lib/validate"
	serviceanalytics "analyticDataCenter/analytics-data-center/internal/services/analytics"
	"encoding/json"
	"log/slog"
	"net/http"
)

type DBHandlers struct {
	log              *slog.Logger
	serviceAnalytics *serviceanalytics.AnalyticsDataCenterService
}

func NewDBHandler(log *slog.Logger, serviceAnalytics *serviceanalytics.AnalyticsDataCenterService) *DBHandlers {
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

func (d *DBHandlers) GetDB(w http.ResponseWriter, r *http.Request) {
	const op = "DBHandlers.GetDB"
	d.log.With(
		slog.String("op", op),
	)
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	var connectionsStrings models.ConnectionStrings
	if err := json.NewDecoder(r.Body).Decode(&connectionsStrings); err != nil {
		d.log.Error("failed to decode request", slog.String("error", err.Error()))
		http.Error(w, "invalid request", http.StatusBadRequest)
	}
	_, err := validate.Validate(connectionsStrings)
	if err != nil {
		d.log.Error("failed validate", slog.String("error", err.Error()))
		http.Error(w, "invalid body", http.StatusBadRequest)
	}
	w.WriteHeader(http.StatusOK)

	dbNames, err := d.serviceAnalytics.GetDB(ctx, connectionsStrings)

	if err := json.NewEncoder(w).Encode(dbNames); err != nil {
		d.log.Error("failed to encode response", slog.String("error", err.Error()))
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

}
