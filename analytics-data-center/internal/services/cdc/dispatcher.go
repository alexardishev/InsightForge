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
	var eventData models.CDCEventData
	if err := json.Unmarshal(eventBytes, &eventData); err != nil {
		log.Printf("Ошибка парсинга JSON: %v", err)
		return
	}

	evt := models.CDCEvent{
		Event: "",
		ID:    "",
		Data:  eventData, // всё остальное — в Data
	}

	analyticsHandler.EventPreprocessing(evt)
}
