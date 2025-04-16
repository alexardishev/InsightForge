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
