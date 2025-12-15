package models

import "time"

// Notification represents a message that can be delivered to UI clients via websockets.
type Notification struct {
	Type      string    `json:"type"`
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	TaskID    string    `json:"taskId,omitempty"`
	Status    string    `json:"status,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}
