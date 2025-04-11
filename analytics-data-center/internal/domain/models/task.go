package models

import "time"

type Task struct {
	ID         string    `json:"id"`
	Status     string    `json:"status"`
	CreateDate time.Time `json:"create_date,omitempty"`
}
