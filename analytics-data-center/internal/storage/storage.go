package storage

import (
	"errors"

	loggerpkg "analyticDataCenter/analytics-data-center/internal/logger"
)

var (
	ErrUserExists         = errors.New("пользователь уже существует")
	ErrUserNotFound       = errors.New("пользователь не найден")
	ErrAppNotFound        = errors.New("приложение не найдено")
	ErrSessionNotFound    = errors.New("сессия не найдена")
	ErrSchemaNotFound     = errors.New("схема не найдена")
	ErrSuggestionNotFound = errors.New("предложение не найдено")
)

type Storage struct {
	DbSys SysDB
	log   *loggerpkg.Logger
	DbDWH DWHDB
}

func New(dbSys SysDB, log *loggerpkg.Logger, dbDWH DWHDB) (*Storage, error) {

	return &Storage{
		DbSys: dbSys,
		log:   log,
		DbDWH: dbDWH,
	}, nil

}
