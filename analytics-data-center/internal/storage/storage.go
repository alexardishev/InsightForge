package storage

import "errors"

var (
	ErrUserExists      = errors.New("пользователь уже существует")
	ErrUserNotFound    = errors.New("пользователь не найден")
	ErrAppNotFound     = errors.New("приложение не найдено")
	ErrSessionNotFound = errors.New("сессия не найдена")
	ErrSchemaNotFound  = errors.New("схема не найдена")
)
