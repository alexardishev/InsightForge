package handlers

import (
	"log/slog"
	"net/http"
)

type Handlers struct {
	log *slog.Logger
	HandlersDB
}

type HandlersDB interface {
	GetConnectionsStrings(w http.ResponseWriter, r *http.Request)
	GetDBInformations(w http.ResponseWriter, r *http.Request)
}

func NewHandlers(log *slog.Logger, db HandlersDB) *Handlers {
	return &Handlers{
		log:        log,
		HandlersDB: db,
	}
}
