package postgres

import (
	"context"
	"log/slog"
)

func (s *Storage) CreateTempTablePostgres(ctx context.Context, query string) error {
	const op = "Storage.PostgreSQL.CreateTempTablePostgres"
	log := s.log.With(
		slog.String("op", op),
		slog.String("query", query),
	)
	log.Info("создание таблицы запущено")

	stmt, err := s.dbDWH.Prepare(query)
	if err != nil {
		log.Error("ошибка подготовки запроса", slog.String("error", err.Error()))
		return err
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx)
	if err != nil {
		log.Error("не удалось удалить таблицу", slog.String("error", err.Error()))
		return err
	}
	return nil
}

func (s *Storage) DeleteTempTablePostgres(ctx context.Context, tableName string) error {
	const op = "Storage.PostgreSQL.DeleteTempTablePostgres"
	log := s.log.With(
		slog.String("op", op),
		slog.String("tableName", tableName),
	)
	log.Info("удаление таблицы запущено")

	query := "DROP TABLE IF EXISTS ($1)"
	stmt, err := s.dbDWH.Prepare(query)
	if err != nil {
		log.Error("ошибка подготовки запроса", slog.String("error", err.Error()))
		return err
	}
	defer stmt.Close()
	_, err = stmt.ExecContext(ctx, tableName)
	if err != nil {
		log.Error("не удалось удалить таблицу", slog.String("error", err.Error()))
		return err
	}
	return nil
}
