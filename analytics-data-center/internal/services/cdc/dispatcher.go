package cdc

import (
	"analyticDataCenter/analytics-data-center/internal/domain/models"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"analyticDataCenter/analytics-data-center/internal/logger"
)

type HandlerCDC interface {
	EventPreprocessing(ctx context.Context, evt models.CDCEvent) error
}

// Dispatch разбирает сообщение Debezium и синхронно передаёт его на обработку.
// nil означает, что событие полностью записано в DWH и offset можно коммитить.
func Dispatch(ctx context.Context, eventBytes []byte, log *logger.Logger, analyticsHandler HandlerCDC) error {
	const op = "Dispatch"
	if len(eventBytes) == 0 {
		// Tombstone после DELETE (tombstones.on.delete=true): данных нет, обрабатывать нечего.
		log.InfoMsg(logger.MsgCDCTombstoneSkipped)
		return nil
	}

	var eventData models.CDCEventData
	if err := json.Unmarshal(eventBytes, &eventData); err != nil {
		log.ErrorMsg(logger.Message{RU: "Ошибка парсинга JSON", EN: "JSON parse error", CN: "JSON解析错误"}, slog.String("error", err.Error()))
		return Permanent(fmt.Errorf("%s: JSON parse error: %w", op, err))
	}

	evt := models.CDCEvent{
		Event: "",
		ID:    "",
		Data:  eventData, // всё остальное — в Data
	}

	if err := analyticsHandler.EventPreprocessing(ctx, evt); err != nil {
		return fmt.Errorf("%s: event processing failed: %w", op, err)
	}
	return nil
}
