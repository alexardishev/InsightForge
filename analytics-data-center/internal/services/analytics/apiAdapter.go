package serviceanalytics

import (
	"analyticDataCenter/analytics-data-center/internal/domain/models"
	"context"
	"fmt"
	"net/url"
)

func (a *AnalyticsDataCenterService) GetDBInformations(ctx context.Context, connections models.ConnectionStrings) ([]models.Source, error) {
	var response []models.Source
	for _, conn := range connections.ConnectionStrings {
		for _, value := range conn.ConnectionString {
			var source models.Source
			connectionString := value
			u, err := url.Parse(connectionString)
			if err != nil {
				a.log.Error("Парсинг строки подключения завершен с ошибкой")
				return nil, fmt.Errorf("ошибка парсинга url: %w", err)
			}
			dbname := u.Path[1:]
			source.Name = dbname
			oltpStorage, err := a.OLTPFactory.GetOLTPStorage(ctx, source.Name)
			if err != nil {
				a.log.Error("Получение хранилища OLTP завершено с ошибкой")
				return nil, fmt.Errorf("ошибка получения хранилища: %w", err)
			}
			schemas, err := oltpStorage.GetSchemas(ctx, source.Name)
			if err != nil {
				a.log.Error("ошибка получения схем")
				return nil, fmt.Errorf("ошибка получения схем: %w", err)
			}
			for sidx, schema := range schemas {
				tables, err := oltpStorage.GetTables(ctx, schema.Name)
				if err != nil {
					a.log.Error("ошибка получения таблиц")
					return nil, fmt.Errorf("ошибка получения таблиц: %w", err)
				}
				for tidx, table := range tables {
					columns, err := oltpStorage.GetColumns(ctx, schema.Name, table.Name)
					if err != nil {
						a.log.Error("ошибка получения столбцов")
						return nil, fmt.Errorf("ошибка получения столбцов: %w", err)
					}
					for cidx, column := range columns {
						info, err := oltpStorage.GetColumnInfo(ctx, table.Name, column.Name)
						if err != nil {
							a.log.Error("ошибка получения информации о столбце")
							return nil, fmt.Errorf("ошибка получения информации о столбце: %w", err)
						}
						columns[cidx].IsPrimaryKey = info.IsPK
						columns[cidx].IsNullable = info.IsNullable
						columns[cidx].Type = info.Type
						columns[cidx].IsFK = info.IsFK
						columns[cidx].IsUNQ = info.IsUnique
						if info.Default != nil {
							columns[cidx].Default = *info.Default
						}
					}
					tables[tidx].Columns = columns
				}
				schemas[sidx].Tables = tables
			}
			source.Schemas = schemas
			response = append(response, source)
		}
	}
	return response, nil
}

func (a *AnalyticsDataCenterService) UploadSchema(ctx context.Context, schema models.View) (int64, error) {
	var id int64
	id, err := a.SchemaProvider.UploadView(ctx, schema)
	if err != nil {
		a.log.Error("Ошибка работы с базой данных")
		return 0, err
	}
	return id, nil

}

func (a *AnalyticsDataCenterService) GetTasks(ctx context.Context, filters models.TaskFilter) ([]models.Task, error) {
	tasks, err := a.TaskService.GetTasks(ctx, filters)
	if err != nil {
		a.log.Error("Ошибка работы с базой данных")
		return nil, err
	}
	return tasks, nil
}
