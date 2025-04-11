package postgres

import (
	"analyticDataCenter/analytics-data-center/internal/domain/models"
	"analyticDataCenter/analytics-data-center/internal/storage"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
)

func (s *Storage) GetView(ctx context.Context, idView int64) (models.View, error) {
	var rowSchema []byte

	const op = "Storage.PostgreSQL.GetSchema"
	log := s.log.With(
		slog.String("op", op),
		slog.Int64("appID", idView),
	)

	log.Info("Operation starting")
	query := "SELECT schema_view FROM schems WHERE schema_id = ($1)"
	err := s.db.QueryRowContext(ctx, query, idView).Scan(&rowSchema)
	if errors.Is(err, sql.ErrNoRows) {
		log.Warn("конфигурация представления не найдена")
		return models.View{}, storage.ErrSessionNotFound
	}
	if err != nil {
		log.Error("Запрос выполнен с ошибкой", slog.String("error", err.Error()))
		return models.View{}, err
	}
	var view models.View
	if err := json.Unmarshal(rowSchema, &view); err != nil {
		log.Error("Ошибка при разборе JSON", slog.String("error", err.Error()))
		return models.View{}, err
	}

	return view, nil

}
