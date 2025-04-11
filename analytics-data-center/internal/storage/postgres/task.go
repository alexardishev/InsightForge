package postgres

import (
	"analyticDataCenter/analytics-data-center/internal/domain/models"
	"context"
	"log/slog"
	"time"
)

func (s *Storage) CreateTask(ctx context.Context, taskID string, status string) error {
	const op = "Storage.PostgreSQL.CreateTask"
	log := s.log.With(
		slog.String("op", op),
		slog.String("taskID", taskID),
	)
	log.Info("создание задачи")

	createDate := time.Now()

	query := "INSERT INTO tasks (id, create_at, status) VALUES ($1, $2, $3)"

	rows, err := s.db.QueryContext(ctx, query, taskID, createDate, status)
	if err != nil {
		log.Error("Запросвыполнен с ошибкой", slog.String("error", err.Error()))
		return err
	}
	rows.Close()

	return nil

}

func (s *Storage) GetTask(ctx context.Context, taskID string) (task models.Task, err error) {
	const op = "Storage.PostgreSQL.CreateTask"
	log := s.log.With(
		slog.String("op", op),
		slog.String("taskID", taskID),
	)
	log.Info("получение задачи")

	query := "SELECT id, create_at, status FROM tasks WHERE id = ($1)"

	err = s.db.QueryRowContext(ctx, query, taskID).Scan(&task.ID, &task.CreateDate, &task.Status)
	if err != nil {
		log.Error("Запросвыполнен с ошибкой", slog.String("error", err.Error()))
		return models.Task{}, err
	}

	return task, nil
}

func (s *Storage) ChangeStatusTask(ctx context.Context, taskID string, newStatus string, comment string) error {
	const op = "Storage.PostgreSQL.CreaChangeStatusTaskteTask"
	log := s.log.With(
		slog.String("op", op),
		slog.String("taskID", taskID),
	)
	log.Info("изменение статуса задачи")

	query := "UPDATE tasks SET status=($1), comment=($2) WHERE id = ($3)"
	_, err := s.db.ExecContext(ctx, query, newStatus, comment, taskID)
	if err != nil {
		log.Error("Запросвыполнен с ошибкой", slog.String("error", err.Error()))
		return err
	}
	return nil

}
