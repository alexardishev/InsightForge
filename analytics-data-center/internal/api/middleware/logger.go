package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

// Logger is a middleware that logs HTTP requests.
type Logger struct {
	log *slog.Logger
}

// NewLogger creates a new instance of Logger middleware using the provided logger.
func NewLogger(logger *slog.Logger) *Logger {
	return &Logger{log: logger}
}

// Middleware returns a chi compatible middleware handler.
func (l *Logger) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sr := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		start := time.Now()

		next.ServeHTTP(sr, r)

		l.log.Info("request completed",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", sr.status),
			slog.Duration("duration", time.Since(start)),
		)
	})
}

// statusRecorder wraps http.ResponseWriter to capture the response status code.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}
