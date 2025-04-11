package tasksserivce

import (
	"analyticDataCenter/analytics-data-center/internal/domain/models"
	"context"
	"errors"
	"log/slog"
	"slices"
)

type TasksService struct {
	log          *slog.Logger
	TaskProvider TaskProvider
	StatusEnum   []string
}

func New(
	log *slog.Logger,
	taskProvider TaskProvider,
	statusEnum []string,

) *TasksService {
	return &TasksService{
		log:          log,
		TaskProvider: taskProvider,
		StatusEnum:   statusEnum,
	}
}

type TaskProvider interface {
	CreateTask(ctx context.Context, taskID string, status string) error
	GetTask(ctx context.Context, taskID string) (models.Task, error)
	ChangeStatusTask(ctx context.Context, taskID string, newStatus string, comment string) error
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
	log.Info("CreateTask start")
	err := s.TaskProvider.CreateTask(ctx, taskID, status)
	if err != nil {
		log.Error("не удалось создать задачу", slog.String("задача", err.Error()))
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
	log.Info("ChangeStatusTask start")
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
		log.Error("не изменить статус у задачи", slog.String("задача", err.Error()))
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
	log.Info("GetTask start")

	task, err := s.TaskProvider.GetTask(ctx, taskID)
	if err != nil {
		log.Error("не изменить статус у задачи", slog.String("задача", err.Error()))
		return models.Task{}, err
	}
	return task, nil
}
