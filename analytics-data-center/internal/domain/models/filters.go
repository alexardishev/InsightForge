package models

import "time"

type TaskFilter struct {
	StartDate *time.Time `json:"start_date,omitempty"` // начало диапазона (опционально)
	EndDate   *time.Time `json:"end_date,omitempty"`   // конец диапазона (опционально)
	Page      int        `json:"page,omitempty"`       // номер страницы
	PageSize  int        `json:"page_size,omitempty"`  // размер страницы
}
