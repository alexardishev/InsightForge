package serviceanalytics

import (
	"analyticDataCenter/analytics-data-center/internal/domain/models"
	"context"
	"fmt"
	"net/url"
)

func (a *AnalyticsDataCenterService) GetDB(ctx context.Context, connections models.ConnectionStrings) ([]string, error) {

	var dbNames []string
	for _, conn := range connections.ConnectionStrings {
		for _, value := range conn.ConnectionString {
			connectionString := value
			u, err := url.Parse(connectionString)
			if err != nil {
				return []string{}, fmt.Errorf("ошибка парсинга url: %w", err)
			}
			dbname := u.Path[1:]
			dbNames = append(dbNames, dbname)

		}
	}
	return dbNames, nil

}
