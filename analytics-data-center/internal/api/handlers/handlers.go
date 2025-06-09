package handlers

import (
	"net/http"

	loggerpkg "analyticDataCenter/analytics-data-center/internal/logger"
)

type Handlers struct {
	log *loggerpkg.Logger
	HandlersDB
}

type HandlersDB interface {
	GetConnectionsStrings(w http.ResponseWriter, r *http.Request)
	GetDB(w http.ResponseWriter, r *http.Request)
}

func NewHandlers(log *loggerpkg.Logger, db HandlersDB) *Handlers {
	return &Handlers{
		log:        log,
		HandlersDB: db,
	}
}
