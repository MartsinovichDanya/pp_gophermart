package logger

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

// Log — глобальный экземпляр логгера.
var Log *zap.Logger = zap.NewNop()

// Initialize инициализирует глобальный логгер с указанным уровнем логирования.
// Принимает строковое представление уровня (DEBUG, INFO, WARN, ERROR).
// Возвращает ошибку, если уровень некорректен или логгер не удалось создать.
func Initialize(level string) error {
	// Преобразуем текстовый уровень логирования в zap.AtomicLevel
	lvl, err := zap.ParseAtomicLevel(level)
	if err != nil {
		return err
	}
	// Создаём новую конфигурацию логера
	cfg := zap.NewProductionConfig()
	// Устанавливаем уровень
	cfg.Level = lvl
	// Создаём логгер на основе конфигурации
	zl, err := cfg.Build()
	if err != nil {
		return err
	}
	// Устанавливаем синглтон
	Log = zl
	defer Log.Sync()
	return nil
}

// GetLogger возвращает middleware для логирования HTTP-запросов.
// Использует chi/middleware.RequestLogger для интеграции с роутером.
func GetLogger() func(next http.Handler) http.Handler {
	return middleware.RequestLogger(&Formatter{Log})
}

// Formatter реализует интерфейс middleware.LogFormatter для zap.
type Formatter struct {
	logger *zap.Logger
}

// LogEntry реализует интерфейс middleware.LogEntry для zap.
type LogEntry struct {
	logger *zap.Logger
	method string
	path   string
}

// NewLogEntry создаёт новую запись лога для HTTP-запроса.
func (f *Formatter) NewLogEntry(r *http.Request) middleware.LogEntry {
	return &LogEntry{
		logger: f.logger,
		method: r.Method,
		path:   r.URL.Path,
	}
}

// Write записывает информацию о завершённом запросе в лог.
func (e *LogEntry) Write(status, bytes int, header http.Header, elapsed time.Duration, extra interface{}) {
	e.logger.Info("request",
		zap.String("method", e.method),
		zap.String("path", e.path),
		zap.Int("status", status),
		zap.Int("bytes", bytes),
		zap.Duration("elapsed", elapsed),
	)
}

// Panic записывает информацию о панике в лог.
func (e *LogEntry) Panic(v interface{}, stack []byte) {
	e.logger.Error("panic",
		zap.Any("panic", v),
		zap.String("stack", string(stack)),
	)
}
