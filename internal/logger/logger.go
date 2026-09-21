package logger

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

// Log — глобальный экземпляр логгера.
var Log *zap.Logger = zap.NewNop()

// Initialize инициализирует глобальный логгер с указанным уровнем логирования.
func Initialize(level string) error {
	lvl, err := zap.ParseAtomicLevel(level)
	if err != nil {
		return err
	}
	cfg := zap.NewProductionConfig()
	cfg.Level = lvl
	zl, err := cfg.Build()
	if err != nil {
		return err
	}
	Log = zl
	defer Log.Sync()
	return nil
}

// responseWriter оборачивает http.ResponseWriter для перехвата статус-кода.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader перехватывает код ответа.
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// GetLogger возвращает middleware для логирования HTTP-запросов.
func GetLogger() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(wrapped, r)
			Log.Info("request",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.Int("status", wrapped.statusCode),
				zap.Duration("elapsed", time.Since(start)),
			)
		})
	}
}
