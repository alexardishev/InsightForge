package cdc

import (
	"analyticDataCenter/analytics-data-center/internal/domain/models"
	"encoding/json"
	"log/slog"

	"analyticDataCenter/analytics-data-center/internal/logger"
)

type HandlerCDC interface {
	EventPreprocessing(models.CDCEvent)
}

func Dispatch(eventBytes []byte, log *logger.Logger, analyticsHandler HandlerCDC) {
	var eventData models.CDCEventData
	if err := json.Unmarshal(eventBytes, &eventData); err != nil {
		log.ErrorMsg(logger.Message{RU: "Ошибка парсинга JSON", EN: "JSON parse error", CN: "JSON解析错误"}, slog.String("error", err.Error()))
		return
	}

	evt := models.CDCEvent{
		Event: "",
		ID:    "",
		Data:  eventData, // всё остальное — в Data
	}

	analyticsHandler.EventPreprocessing(evt)
}
