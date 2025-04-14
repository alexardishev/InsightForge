package storage

import (
	"errors"
	"log/slog"
)

var (
	ErrUserExists      = errors.New("пользователь уже существует")
	ErrUserNotFound    = errors.New("пользователь не найден")
	ErrAppNotFound     = errors.New("приложение не найдено")
	ErrSessionNotFound = errors.New("сессия не найдена")
	ErrSchemaNotFound  = errors.New("схема не найдена")
)

type Storage struct {
	DbSys SysDB
	log   *slog.Logger
	DbDWH DWHDB
}

func New(dbSys SysDB, log *slog.Logger, dbDWH DWHDB) (*Storage, error) {

	return &Storage{
		DbSys: dbSys,
		log:   log,
		DbDWH: dbDWH,
	}, nil

}
