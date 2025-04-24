package postgresdwh

import (
	"context"
	"log/slog"
)

func (p *PostgresDWH) InsertDataToDWH(ctx context.Context, query string) error {
	const op = "Storage.PostgreSQL.InsertDataToDWH"
	log := p.Log.With(
		slog.String("op", op),
	)
	_, err := p.Db.ExecContext(ctx, query)
	if err != nil {
		log.Error("ошибка подготовки запроса", slog.String("error", err.Error()))
		return err
	}

	return nil
}
