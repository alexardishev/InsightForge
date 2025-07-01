package logger

import (
	"log/slog"
	"os"
)

type Message struct {
	RU string
	EN string
	CN string
}

type Logger struct {
	*slog.Logger
	lang string
}

func New(env, lang string) *Logger {
	var base *slog.Logger
	switch env {
	case "development":
		base = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	case "production":
		base = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	default:
		base = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}
	return &Logger{Logger: base, lang: lang}
}

func (l *Logger) selectMsg(msg Message) string {
	switch l.lang {
	case "en":
		if msg.EN != "" {
			return msg.EN
		}
	case "cn":
		if msg.CN != "" {
			return msg.CN
		}
	}
	if msg.RU != "" {
		return msg.RU
	}
	if msg.EN != "" {
		return msg.EN
	}
	return msg.CN
}

func (l *Logger) InfoMsg(msg Message, args ...any) {
	l.Logger.Info(l.selectMsg(msg), args...)
}

func (l *Logger) ErrorMsg(msg Message, args ...any) {
	l.Logger.Error(l.selectMsg(msg), args...)
}

func (l *Logger) DebugMsg(msg Message, args ...any) {
	l.Logger.Debug(l.selectMsg(msg), args...)
}

func (l *Logger) WarnMsg(msg Message, args ...any) {
	l.Logger.Warn(l.selectMsg(msg), args...)
}

func (l *Logger) With(args ...any) *Logger {
	return &Logger{Logger: l.Logger.With(args...), lang: l.lang}
}
