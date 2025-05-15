package serviceanalytics

import "analyticDataCenter/analytics-data-center/internal/services/cdc"

func (a *AnalyticsDataCenterService) HandleCDCEvent(evt cdc.CDCEvent) {
	a.log.Info("Пришло событие из кафка")
}
