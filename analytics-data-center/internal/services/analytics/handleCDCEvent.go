package serviceanalytics

import (
	"analyticDataCenter/analytics-data-center/internal/domain/models"
	"log/slog"
)

func (a *AnalyticsDataCenterService) eventWorker() {
	for event := range a.eventQueue {
		log := a.log.With(
			slog.String("component", "EventWorker"),
		)
		log.Info("Событие пришло в eventWorker")
		a.handlerCDCFunc(event)

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

func (a *AnalyticsDataCenterService) handlerCDCFunc(evt models.CDCEvent) {
	const op = "handlerCDCFunc"
	log := a.log.With(
		slog.String("op", op),
		slog.String("eventId", evt.ID),
	)
	log.Info("Пришло событие из кафка", slog.Any("", evt.Data))
}
