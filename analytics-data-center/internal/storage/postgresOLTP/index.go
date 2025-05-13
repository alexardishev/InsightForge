package postgresoltp

import (
	"analyticDataCenter/analytics-data-center/internal/domain/models"
	"context"
	"log/slog"
)

func (p *PostgresOLTP) GetIndexes(ctx context.Context, tableName string, schemaName string) (models.Indexes, error) {
	const op = "Storage.PostgresOLTP.GetIndexes"
	log := p.Log.With(
		slog.String("op", op),
		slog.String("tableName", tableName),
	)

	var indexes models.Indexes

	query := "SELECT indexname,indexdef FROM pg_indexes WHERE schemaname = $1 AND tablename = $2;"

	rows, err := p.Db.QueryContext(ctx, query, schemaName, tableName)
	if err != nil {
		log.Error("ошибка при запросе индексов", slog.String("ошибка", err.Error()))
		return models.Indexes{}, err
	}

	defer rows.Close()

	for rows.Next() {
		var index models.Index
		err = rows.Scan(&index.IndexName, &index.IndexDef)
		if err != nil {
			log.Error("ошибка сканирования", slog.String("ошибка", err.Error()))
			return models.Indexes{}, err
		}
		indexes.Indexes = append(indexes.Indexes, index)
	}

	if err := rows.Err(); err != nil {
		log.Error("ошибка после сканирования всех строк", slog.String("ошибка", err.Error()))
		return models.Indexes{}, err
	}
	return indexes, nil

}
