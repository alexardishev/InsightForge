package dbhandlers

import (
	"analyticDataCenter/analytics-data-center/internal/domain/models"
	"analyticDataCenter/analytics-data-center/internal/lib/validate"
	loggerpkg "analyticDataCenter/analytics-data-center/internal/logger"
	serviceanalytics "analyticDataCenter/analytics-data-center/internal/services/analytics"
	"analyticDataCenter/analytics-data-center/internal/storage"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
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

	var req models.DBInfoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		d.log.Error("failed to decode request", slog.String("error", err.Error()))
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	_, err := validate.Validate(req)
	if err != nil {
		d.log.Error("failed validate", slog.String("error", err.Error()))
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	dbNames, err := d.serviceAnalytics.GetDBInformations(ctx, req.ConnectionStrings, req.Page, req.PageSize)
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

func (d *DBHandlers) GetColumnMismatchGroups(w http.ResponseWriter, r *http.Request) {
	const op = "DBHandlers.GetColumnMismatchGroups"
	log := d.log.With(slog.String("op", op))

	ctx := r.Context()
	query := r.URL.Query()
	filter := models.ColumnMismatchFilter{Limit: 50, Offset: 0}

	if schemaIDStr := query.Get("schemaId"); schemaIDStr != "" {
		if id, err := strconv.ParseInt(schemaIDStr, 10, 64); err == nil {
			filter.SchemaID = &id
		}
	}

	if database := query.Get("database"); database != "" {
		filter.DatabaseName = &database
	}

	if schema := query.Get("schema"); schema != "" {
		filter.SchemaName = &schema
	}

	if table := query.Get("table"); table != "" {
		filter.TableName = &table
	}

	if status := query.Get("status"); status != "" {
		filter.Status = &status
	}

	if limitStr := query.Get("limit"); limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil {
			filter.Limit = v
		}
	}

	if offsetStr := query.Get("offset"); offsetStr != "" {
		if v, err := strconv.Atoi(offsetStr); err == nil {
			filter.Offset = v
		}
	}

	groups, err := d.serviceAnalytics.ListColumnMismatchGroups(ctx, filter)
	if err != nil {
		log.Error("failed to list mismatch groups", slog.String("error", err.Error()))
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"items":  groups,
		"limit":  filter.Limit,
		"offset": filter.Offset,
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Error("failed to encode response", slog.String("error", err.Error()))
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
}

func (d *DBHandlers) GetColumnMismatchGroup(w http.ResponseWriter, r *http.Request) {
	const op = "DBHandlers.GetColumnMismatchGroup"
	log := d.log.With(slog.String("op", op))

	ctx := r.Context()
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	group, err := d.serviceAnalytics.GetColumnMismatchGroup(ctx, id)
	if err != nil {
		if errors.Is(err, storage.ErrMismatchNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		log.Error("failed to get mismatch group", slog.String("error", err.Error()))
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(group); err != nil {
		log.Error("failed to encode response", slog.String("error", err.Error()))
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
}

func (d *DBHandlers) ApplyColumnMismatchGroup(w http.ResponseWriter, r *http.Request) {
	const op = "DBHandlers.ApplyColumnMismatchGroup"
	log := d.log.With(slog.String("op", op))

	ctx := r.Context()
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var req models.ColumnMismatchResolution
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	if err := d.serviceAnalytics.ApplyColumnMismatchResolution(ctx, id, req); err != nil {
		if errors.Is(err, storage.ErrMismatchNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		log.Error("failed to apply mismatch resolution", slog.String("error", err.Error()))
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
		log.Error("failed to encode response", slog.String("error", err.Error()))
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
}

func (d *DBHandlers) GetColumnRenameSuggestions(w http.ResponseWriter, r *http.Request) {
	const op = "DBHandlers.GetColumnRenameSuggestions"
	d.log.With(slog.String("op", op))

	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	query := r.URL.Query()
	filter := models.ColumnRenameSuggestionFilter{}

	if schemaIDStr := query.Get("schemaId"); schemaIDStr != "" {
		schemaID, err := strconv.ParseInt(schemaIDStr, 10, 64)
		if err != nil {
			d.log.Error("invalid schemaId", slog.String("error", err.Error()))
			http.Error(w, "invalid schemaId", http.StatusBadRequest)
			return
		}
		filter.SchemaID = &schemaID
	}

	if database := query.Get("database"); database != "" {
		filter.DatabaseName = &database
	}

	if schema := query.Get("schema"); schema != "" {
		filter.SchemaName = &schema
	}

	if table := query.Get("table"); table != "" {
		filter.TableName = &table
	}

	limit := 50
	if limitStr := query.Get("limit"); limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil && v > 0 {
			limit = v
		}
	}
	filter.Limit = limit

	if offsetStr := query.Get("offset"); offsetStr != "" {
		if v, err := strconv.Atoi(offsetStr); err == nil && v >= 0 {
			filter.Offset = v
		}
	}

	sortParam := strings.ToLower(query.Get("sort"))
	filter.SortByCreatedAtDesc = sortParam == "" || sortParam == "created_at_desc"
	if sortParam == "created_at_asc" {
		filter.SortByCreatedAtDesc = false
	}

	suggestions, err := d.serviceAnalytics.ListColumnRenameSuggestions(ctx, filter)
	if err != nil {
		d.log.Error("failed to list suggestions", slog.String("error", err.Error()))
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"items":  suggestions,
		"limit":  filter.Limit,
		"offset": filter.Offset,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		d.log.Error("failed to encode response", slog.String("error", err.Error()))
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
}

func (d *DBHandlers) AcceptColumnRenameSuggestion(w http.ResponseWriter, r *http.Request) {
	const op = "DBHandlers.AcceptColumnRenameSuggestion"
	log := d.log.With(slog.String("op", op))

	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		log.Error("invalid suggestion id", slog.String("error", err.Error()))
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	if err := d.serviceAnalytics.AcceptColumnRenameSuggestion(ctx, id); err != nil {
		if errors.Is(err, storage.ErrSuggestionNotFound) {
			http.Error(w, "suggestion not found", http.StatusNotFound)
			return
		}
		log.Error("failed to accept suggestion", slog.String("error", err.Error()))
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
		log.Error("failed to encode response", slog.String("error", err.Error()))
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
}

func (d *DBHandlers) RejectColumnRenameSuggestion(w http.ResponseWriter, r *http.Request) {
	const op = "DBHandlers.RejectColumnRenameSuggestion"
	log := d.log.With(slog.String("op", op))

	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		log.Error("invalid suggestion id", slog.String("error", err.Error()))
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	if err := d.serviceAnalytics.RejectColumnRenameSuggestion(ctx, id); err != nil {
		if errors.Is(err, storage.ErrSuggestionNotFound) {
			http.Error(w, "suggestion not found", http.StatusNotFound)
			return
		}
		log.Error("failed to reject suggestion", slog.String("error", err.Error()))
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
		log.Error("failed to encode response", slog.String("error", err.Error()))
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
}
