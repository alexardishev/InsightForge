package postgresoltp

import (
	"context"
	"log/slog"
)

func (p *PostgresOLTP) GetCountInsertData(ctx context.Context, query string) (int64, error) {
	const op = "Storage.PostgresOLTP.GetCountInsertData"
	var count int64
	log := p.Log.With(
		slog.String("op", op),
		slog.String("query", query),
	)
	log.Info("подсчет количества записей для вставки")
	err := p.Db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		log.Error("ошибка получения количества записей", slog.String("ошибка", err.Error()))
		return 0, err
	}

	return count, nil

}

func (p *PostgresOLTP) SelectDataToInsert(ctx context.Context, query string) ([]map[string]interface{}, error) {
	const op = "Storage.PostgresOLTP.SelectDataToInsert"
	log := p.Log.With(
		slog.String("op", op),
		slog.String("query", query),
	)
	log.Info("выборка данных для вставки")

	rows, err := p.Db.QueryContext(ctx, query)
	if err != nil {
		log.Error("ошибка получения данных для вставки", slog.String("ошибка", err.Error()))
		return nil, err
	}
	defer rows.Close()
	columns, err := rows.Columns()
	results := make([]map[string]interface{}, 0)
	if err != nil {
		log.Error("ошибка получения колонок", slog.String("ошибка", err.Error()))
		return nil, err
	}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuesPointers := make([]interface{}, len(columns))

		for i := range values {
			valuesPointers[i] = &values[i]
		}
		if err := rows.Scan(valuesPointers...); err != nil {
			log.Error("ошибка сканирования строк", slog.String("ошибка", err.Error()))
		}
		rowMap := make(map[string]interface{})
		for i, col := range columns {
			rowMap[col] = values[i]
		}

		results = append(results, rowMap)

	}
	return results, nil
}
