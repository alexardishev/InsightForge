package handlers

import (
	"net/http"

	loggerpkg "analyticDataCenter/analytics-data-center/internal/logger"
)

type Handlers struct {
	log *loggerpkg.Logger
	HandlersDB
	HandlersTasks
}

type HandlersDB interface {
	GetConnectionsStrings(w http.ResponseWriter, r *http.Request)
	GetDBInformations(w http.ResponseWriter, r *http.Request)
	UploadSchema(w http.ResponseWriter, r *http.Request)
	GetColumnRenameSuggestions(w http.ResponseWriter, r *http.Request)
}

type HandlersTasks interface {
	GetTasks(w http.ResponseWriter, r *http.Request)
}

func NewHandlers(log *loggerpkg.Logger, db HandlersDB, tasks HandlersTasks) *Handlers {
	return &Handlers{
		log:           log,
		HandlersDB:    db,
		HandlersTasks: tasks,
	}
}
