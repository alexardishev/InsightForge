package serviceanalytics

import (
	"analyticDataCenter/analytics-data-center/internal/domain/models"
	"context"
	"log/slog"
)

func (a *AnalyticsDataCenterService) createRowAfterListenEventInDWH(ctx context.Context, evtData models.CDCEventData) (string, error) {
	const op = "createRowAfterListenEventInDWH"

	log := a.log.With(
		slog.String("op", op),
	)

	databaseEvt := evtData.Source.DB
	schemaEvt := evtData.Source.Schema
	tableEvt := evtData.Source.Table

	log.Info("Начинаю создание новой строки", slog.Any("", evtData.Op))
	schemaIds, err := a.SchemaProvider.GetSchems(ctx, databaseEvt, schemaEvt, tableEvt)
	if err != nil {
		log.Error("ошибка получения схемы", slog.Any("ошибка", err))
		return "", err
	}
	log.Info("схемы", slog.Any("", schemaIds))
	return "", nil
}
