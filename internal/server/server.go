package server

import (
	"net/http"

	"github.com/MartsinovichDanya/pp_gophermart/internal/auth"
	"github.com/MartsinovichDanya/pp_gophermart/internal/config"
	"github.com/MartsinovichDanya/pp_gophermart/internal/handler"
	"github.com/MartsinovichDanya/pp_gophermart/internal/logger"
	"github.com/MartsinovichDanya/pp_gophermart/internal/storage"
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

	// Инициализация хранилища
	store, err := storage.NewPostgresStore(cfg.DatabaseURI)
	if err != nil {
		logger.Log.Fatal("Failed to initialize storage", zap.Error(err))
		return err
	}
	defer store.Close()
	logger.Log.Info("Storage initialized successfully")

	// Инициализация handler
	h := handler.NewHandler(store, cfg.JWTSecret, cfg.MaxBodySize)

	r := chi.NewRouter()
	r.Use(chiMiddleware.RequestID)
	r.Use(logger.GetLogger())
	r.Use(chiMiddleware.Recoverer)

	// Публичные маршруты (без авторизации)
	r.Post("/api/user/register", h.RegisterHandler)
	r.Post("/api/user/login", h.LoginHandler)

	// Защищённые маршруты (требуют авторизации)
	r.Group(func(r chi.Router) {
		r.Use(auth.AuthMiddleware(cfg.JWTSecret))

		// Заказы
		r.Post("/api/user/orders", h.UploadOrderHandler)
		r.Get("/api/user/orders", h.GetOrdersHandler)

		// TODO: Добавить маршруты баланса и выводов (этап 6)
	})

	// Health-check эндпоинт
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
