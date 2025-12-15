package notifications

import (
	"analyticDataCenter/analytics-data-center/internal/domain/models"
	loggerpkg "analyticDataCenter/analytics-data-center/internal/logger"
	"log/slog"
	"net/http"

	"github.com/gorilla/websocket"
)

// Notifier defines behaviour for publishing notifications.
type Notifier interface {
	Publish(notification models.Notification)
}

// Worker receives notifications from internal components and broadcasts them to websocket clients.
type Worker struct {
	log        *loggerpkg.Logger
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	clients    map[*websocket.Conn]bool
	events     chan models.Notification
	upgrader   websocket.Upgrader
}

// NewWorker constructs notification worker and starts background routines.
func NewWorker(log *loggerpkg.Logger) *Worker {
	worker := &Worker{
		log:        log,
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
		clients:    make(map[*websocket.Conn]bool),
		events:     make(chan models.Notification, 100),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}

	go worker.run()
	return worker
}

// Publish enqueues notification to be broadcasted.
func (w *Worker) Publish(notification models.Notification) {
	select {
	case w.events <- notification:
	default:
		w.log.Warn("notification channel is full, dropping message", slog.String("type", notification.Type))
	}
}

// HandleConnection upgrades incoming HTTP request to websocket and registers client.
func (w *Worker) HandleConnection(wr http.ResponseWriter, r *http.Request) {
	conn, err := w.upgrader.Upgrade(wr, r, nil)
	if err != nil {
		w.log.Error("failed to upgrade connection", slog.String("error", err.Error()))
		return
	}

	w.register <- conn

	go func() {
		defer func() {
			w.unregister <- conn
			conn.Close()
		}()

		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				break
			}
		}
	}()
}

func (w *Worker) run() {
	for {
		select {
		case conn := <-w.register:
			w.clients[conn] = true
			w.log.Info("websocket client registered", slog.Int("clients", len(w.clients)))
		case conn := <-w.unregister:
			if _, ok := w.clients[conn]; ok {
				delete(w.clients, conn)
				w.log.Info("websocket client unregistered", slog.Int("clients", len(w.clients)))
			}
		case event := <-w.events:
			for client := range w.clients {
				if err := client.WriteJSON(event); err != nil {
					w.log.Warn("failed to write notification", slog.String("error", err.Error()))
					client.Close()
					delete(w.clients, client)
				}
			}
		}
	}
}
