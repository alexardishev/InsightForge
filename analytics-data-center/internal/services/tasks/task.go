package tasksserivce

import (
	"analyticDataCenter/analytics-data-center/internal/domain/models"
	"analyticDataCenter/analytics-data-center/internal/storage"
	"context"
	"errors"
	"log/slog"
	"slices"

	loggerpkg "analyticDataCenter/analytics-data-center/internal/logger"
)

type TasksService struct {
	log          *loggerpkg.Logger
	TaskProvider storage.SysDB
	StatusEnum   []string
}

func New(
	log *loggerpkg.Logger,
	taskProvider storage.SysDB,
	statusEnum []string,

) *TasksService {
	return &TasksService{
		log:          log,
		TaskProvider: taskProvider,
		StatusEnum:   statusEnum,
	}
}

func (s *TasksService) CreateTask(ctx context.Context, taskID string, status string) error {
	const op = "tasks.CreateTask"
	log := s.log.With(
		slog.String("op", op),
		slog.String("taskID", taskID),
	)
	if taskID == "" {
		return errors.New("идентификатор задачи не может быть пустым")
	}

	if status == "" {
		return errors.New("статус задачи не может быть пустым")
	}
	if !slices.Contains(s.StatusEnum, status) {
		return errors.New("статус не распознан. проверьте правильность указания статуса")
	}
	log.InfoMsg(loggerpkg.MsgCreateTaskStart)
	err := s.TaskProvider.CreateTask(ctx, taskID, status)
	if err != nil {
		log.ErrorMsg(loggerpkg.MsgCreateTaskFailed, slog.String("error", err.Error()))
		return err
	}

	return nil
}

func (s *TasksService) ChangeStatusTask(ctx context.Context, taskID string, newStatus string, comment string) error {
	const op = "tasks.ChangeStatusTask"
	log := s.log.With(
		slog.String("op", op),
		slog.String("taskID", taskID),
	)
	log.InfoMsg(loggerpkg.MsgChangeStatusStart)
	if taskID == "" {
		return errors.New("идентификатор задачи не может быть пустым")
	}

	if newStatus == "" {
		return errors.New("статус задачи не может быть пустым")
	}
	if !slices.Contains(s.StatusEnum, newStatus) {
		return errors.New("статус не распознан. проверьте правильность указания статуса")
	}
	err := s.TaskProvider.ChangeStatusTask(ctx, taskID, newStatus, comment)
	if err != nil {
		log.ErrorMsg(loggerpkg.MsgChangeStatusFailed, slog.String("error", err.Error()))
		return err
	}
	return nil
}

func (s *TasksService) GetTask(ctx context.Context, taskID string) (models.Task, error) {
	const op = "tasks.GetTask"
	log := s.log.With(
		slog.String("op", op),
		slog.String("taskID", taskID),
	)
	log.InfoMsg(loggerpkg.MsgGetTaskStart)

	task, err := s.TaskProvider.GetTask(ctx, taskID)
	if err != nil {
		log.ErrorMsg(loggerpkg.MsgGetTaskFailed, slog.String("error", err.Error()))
		return models.Task{}, err
	}
	return task, nil
}

func (s *TasksService) GetTasks(ctx context.Context, filters models.TaskFilter) ([]models.Task, error) {
	const op = "tasks.GetTasks"
	log := s.log.With(
		slog.String("op", op),
	)
	log.InfoMsg(loggerpkg.MsgGetTaskStart)

	tasks, err := s.TaskProvider.GetTasks(ctx, filters)
	if err != nil {
		log.ErrorMsg(loggerpkg.MsgGetTaskFailed, slog.String("error", err.Error()))
		return nil, err
	}
	return tasks, nil
}
