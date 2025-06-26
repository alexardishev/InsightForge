package postgres

import (
	"analyticDataCenter/analytics-data-center/internal/domain/models"
	"context"
	"log/slog"
	"time"
)

func (p *PostgresSys) CreateTask(ctx context.Context, taskID string, status string) error {
	const op = "Storage.PostgreSQL.CreateTask"
	log := p.Log.With(
		slog.String("op", op),
		slog.String("taskID", taskID),
	)
	log.Info("создание задачи")

	createDate := time.Now()

	query := "INSERT INTO tasks (id, create_at, status) VALUES ($1, $2, $3)"

	_, err := p.Db.ExecContext(ctx, query, taskID, createDate, status)
	if err != nil {
		log.Error("Запросвыполнен с ошибкой", slog.String("error", err.Error()))
		return err
	}
	return nil

}

func (p *PostgresSys) GetTask(ctx context.Context, taskID string) (task models.Task, err error) {
	const op = "Storage.PostgreSQL.CreateTask"
	log := p.Log.With(
		slog.String("op", op),
		slog.String("taskID", taskID),
	)
	log.Info("получение задачи")

	query := "SELECT id, create_at, status FROM tasks WHERE id = ($1)"

	err = p.Db.QueryRowContext(ctx, query, taskID).Scan(&task.ID, &task.CreateDate, &task.Status)
	if err != nil {
		log.Error("Запросвыполнен с ошибкой", slog.String("error", err.Error()))
		return models.Task{}, err
	}

	return task, nil
}

func (p *PostgresSys) ChangeStatusTask(ctx context.Context, taskID string, newStatus string, comment string) error {
	const op = "Storage.PostgreSQL.CreaChangeStatusTaskteTask"
	log := p.Log.With(
		slog.String("op", op),
		slog.String("taskID", taskID),
	)
	log.Info("изменение статуса задачи")

	query := "UPDATE tasks SET status=($1), comment=($2) WHERE id = ($3)"
	_, err := p.Db.ExecContext(ctx, query, newStatus, comment, taskID)
	if err != nil {
		log.Error("Запросвыполнен с ошибкой", slog.String("error", err.Error()))
		return err
	}
	return nil

}

func (p *PostgresSys) GetTasks(ctx context.Context, filters models.TaskFilter) ([]models.Task, error) {
	var tasks []models.Task
	const op = "Storage.PostgreSQL.CreaChangeStatusTaskteTask"
	log := p.Log.With(
		slog.String("op", op),
	)

	if filters.Page < 1 {
		filters.Page = 1
	}
	if filters.PageSize <= 0 {
		filters.PageSize = 10
	}
	log.Info("получение списка задач")
	query := `SELECT id, create_at, status, comment FROM tasks
				WHERE ($1::timestamp IS NULL OR create_at >= $1::timestamp)
	  			AND ($2::timestamp IS NULL OR create_at <= $2::timestamp)
				ORDER BY create_at DESC
				LIMIT $3 OFFSET $4`
	rows, err := p.Db.QueryContext(ctx, query, filters.StartDate, filters.EndDate, filters.PageSize, (filters.Page-1)*filters.PageSize)
	if err != nil {
		log.Error("Запросвыполнен с ошибкой", slog.String("error", err.Error()))
		return []models.Task{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var task models.Task
		err = rows.Scan(&task.ID, &task.CreateDate, &task.Status, &task.Comment)
		if err != nil {
			log.Error("Запросвыполнен с ошибкой", slog.String("error", err.Error()))
			return []models.Task{}, err
		}
		tasks = append(tasks, task)

	}
	return tasks, nil
}
