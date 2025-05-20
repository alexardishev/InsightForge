package serviceanalytics

import (
	"analyticDataCenter/analytics-data-center/internal/domain/models"
	"context"
	"log/slog"
)

func (a *AnalyticsDataCenterService) eventWorker() {
	for event := range a.eventQueue {
		log := a.log.With(
			slog.String("component", "EventWorker"),
		)
		log.Info("Событие пришло в eventWorker")
		ctx := context.Background()
		err := a.handlerCDCFunc(ctx, event)
		if err != nil {
			log.Error("Ошибка при выполнени eventWorker", slog.String("error", err.Error()))
		}
	}
}
func (a *AnalyticsDataCenterService) EventPreprocessing(evt models.CDCEvent) {
	const op = "EventPreprocessing"
	log := a.log.With(
		slog.String("op", op),
		slog.String("eventId", evt.ID),
	)

	log.Info("Пришло событие из Kafka, пересылаю в канал")
	a.eventQueue <- evt
}

func (a *AnalyticsDataCenterService) handlerCDCFunc(ctx context.Context, evt models.CDCEvent) error {
	const op = "handlerCDCFunc"
	log := a.log.With(
		slog.String("op", op),
		slog.String("eventId", evt.ID),
	)
	log.Info("Пришло событие из кафка", slog.Any("", evt.Data))
	eventType := a.eventIdentifier(evt.Data.Op)
	res, err := a.eventDispathFunction(ctx, eventType, evt.Data)
	if err != nil {
		log.Error("Ошибка при выполнении определения функции вызова", slog.String("error", err.Error()))
		return err
	}
	log.Info("Обновления выполнены", slog.String("type", res))
	return nil

}
func (a *AnalyticsDataCenterService) eventIdentifier(typeEvt string) string {
	const op = "eventIdentifier"
	log := a.log.With(
		slog.String("op", op),
		slog.String("eventId", typeEvt),
	)
	log.Info("Определяю тип события", slog.Any("", typeEvt))

	switch typeEvt {
	case "c":
		return "c"
	case "u":
		return "u"
	case "d":
		return "d"
	case "r":
		return "r"
	}
	return ""
}

func (a *AnalyticsDataCenterService) eventDispathFunction(ctx context.Context, typeEvt string, eventData models.CDCEventData) (string, error) {
	const op = "eventDispathFunction"
	log := a.log.With(
		slog.String("op", op),
		slog.String("eventId", typeEvt),
	)
	log.Info("Определяю тип вызываемой функции", slog.Any("", typeEvt))
	switch typeEvt {
	case "c":
		err := a.createRowAfterListenEventInDWH(ctx, eventData)
		if err != nil {
			return "", err
		}
		return "", nil
	case "u":
		return "u", nil
	case "d":
		return "d", nil
	case "r":
		return "r", nil
	}
	return "", nil
}
