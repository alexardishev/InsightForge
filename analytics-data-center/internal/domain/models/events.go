package models

type CDCEvent struct {
	Event string                 `json:"event"`
	ID    string                 `json:"id"`
	Data  map[string]interface{} `json:"data"`
}
