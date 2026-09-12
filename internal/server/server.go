package server

import (
	"net/http"

	"github.com/MartsinovichDanya/pp_gophermart/internal/config"
	"github.com/MartsinovichDanya/pp_gophermart/internal/logger"
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

// Run инициализирует все зависимости, настраивает роутер и запускает HTTP-сервер.
func Run() error {
	cfg := config.GetConfig()

	if err := logger.Initialize(cfg.LogLevel); err != nil {
		return err
	}
	logger.Log.Debug("Running config", zap.Any("config", cfg))

	r := chi.NewRouter()
	r.Use(chiMiddleware.RequestID)
	r.Use(logger.GetLogger())
	r.Use(chiMiddleware.Recoverer)

	// TODO: Добавить маршруты (этапы 4-6)

	// Простой health-check эндпоинт
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Gophermart is running"))
	})

	logger.Log.Info("Starting server", zap.String("address", cfg.RunAddress))
	if err := http.ListenAndServe(cfg.RunAddress, r); err != nil {
		return err
	}

	return nil
}
