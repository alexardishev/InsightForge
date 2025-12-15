package handlers

import (
	"net/http"

	loggerpkg "analyticDataCenter/analytics-data-center/internal/logger"
)

type Handlers struct {
	log *loggerpkg.Logger
	HandlersDB
	HandlersTasks
	HandlersNotifications
}

type HandlersDB interface {
	GetConnectionsStrings(w http.ResponseWriter, r *http.Request)
	GetDBInformations(w http.ResponseWriter, r *http.Request)
	UploadSchema(w http.ResponseWriter, r *http.Request)
	ListViews(w http.ResponseWriter, r *http.Request)
	StartETL(w http.ResponseWriter, r *http.Request)
	GetColumnRenameSuggestions(w http.ResponseWriter, r *http.Request)
	AcceptColumnRenameSuggestion(w http.ResponseWriter, r *http.Request)
	RejectColumnRenameSuggestion(w http.ResponseWriter, r *http.Request)
	GetColumnMismatchGroups(w http.ResponseWriter, r *http.Request)
	GetColumnMismatchGroup(w http.ResponseWriter, r *http.Request)
	ApplyColumnMismatchGroup(w http.ResponseWriter, r *http.Request)
}

type HandlersTasks interface {
	GetTasks(w http.ResponseWriter, r *http.Request)
}

type HandlersNotifications interface {
	NotificationsWS(w http.ResponseWriter, r *http.Request)
}

func NewHandlers(log *loggerpkg.Logger, db HandlersDB, tasks HandlersTasks, notifications HandlersNotifications) *Handlers {
	return &Handlers{
		log:                   log,
		HandlersDB:            db,
		HandlersTasks:         tasks,
		HandlersNotifications: notifications,
	}
}
