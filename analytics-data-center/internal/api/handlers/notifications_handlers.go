package handlers

import (
	"analyticDataCenter/analytics-data-center/internal/notifications"
	"net/http"
)

type NotificationHandlers struct {
	worker *notifications.Worker
}

func NewNotificationHandlers(worker *notifications.Worker) *NotificationHandlers {
	return &NotificationHandlers{worker: worker}
}

func (n *NotificationHandlers) NotificationsWS(w http.ResponseWriter, r *http.Request) {
	n.worker.HandleConnection(w, r)
}
