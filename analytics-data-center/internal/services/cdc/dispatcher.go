package cdc

import (
	"analyticDataCenter/analytics-data-center/internal/domain/models"
	"encoding/json"
	"log"
)

type HandlerCDC interface {
	EventPreprocessing(models.CDCEvent)
}

func Dispatch(eventBytes []byte, analyticsHandler HandlerCDC) {
	var raw map[string]interface{}
	if err := json.Unmarshal(eventBytes, &raw); err != nil {
		log.Printf("Ошибка парсинга JSON: %v", err)
		return
	}

	event, _ := raw["event"].(string)
	id, _ := raw["id"].(string)
	delete(raw, "event")
	delete(raw, "id")

	evt := models.CDCEvent{
		Event: event,
		ID:    id,
		Data:  raw, // всё остальное — в Data
	}

	analyticsHandler.EventPreprocessing(evt)
}
