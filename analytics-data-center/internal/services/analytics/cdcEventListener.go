package serviceanalytics

import (
	"analyticDataCenter/analytics-data-center/internal/domain/models"
	loggerpkg "analyticDataCenter/analytics-data-center/internal/logger"
	"context"
	"log/slog"
)

// EventPreprocessing синхронно применяет CDC-событие к DWH. Возврат nil означает,
// что запись во все вьюхи завершена и cdc.Listener может коммитить offset.
// ctx отменяется при остановке приложения — незавершённая запись откатывается,
// offset не коммитится, и после рестарта событие будет обработано заново.
func (a *AnalyticsDataCenterService) EventPreprocessing(ctx context.Context, evt models.CDCEvent) error {
	const op = "EventPreprocessing"
	log := a.log.With(
		slog.String("op", op),
		slog.String("eventId", evt.ID),
	)

	log.InfoMsg(loggerpkg.MsgForwardCDCEvent)
	return a.handlerCDCFunc(ctx, evt)
}

func (a *AnalyticsDataCenterService) handlerCDCFunc(ctx context.Context, evt models.CDCEvent) error {
	const op = "handlerCDCFunc"
	log := a.log.With(
		slog.String("op", op),
		slog.String("eventId", evt.ID),
	)
	eventType := a.eventIdentifier(evt.Data.Op)
	res, err := a.eventDispathFunction(ctx, eventType, evt.Data)
	if err != nil {
		log.ErrorMsg(loggerpkg.MsgEventHandlerError, slog.String("error", err.Error()))
		return err
	}
	log.InfoMsg(loggerpkg.MsgUpdatesDone, slog.String("type", res))
	return nil

}
func (a *AnalyticsDataCenterService) eventIdentifier(typeEvt string) string {
	const op = "eventIdentifier"
	log := a.log.With(
		slog.String("op", op),
		slog.String("eventId", typeEvt),
	)
	log.InfoMsg(loggerpkg.MsgDeterminingEventType, slog.Any("type", typeEvt))

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
	log.InfoMsg(loggerpkg.MsgDeterminingFuncType, slog.Any("type", typeEvt))
	switch typeEvt {
	case "c":
		err := a.createRowAfterListenEventInDWH(ctx, eventData)
		if err != nil {
			return "", err
		}
		return "", nil
	case "u":
		err := a.createRowAfterListenEventInDWH(ctx, eventData)
		if err != nil {
			return "", err
		}
		return "u", nil
	case "d":
		return "d", nil
	case "r":
		return "r", nil
	}
	return "", nil
}
