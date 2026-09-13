package server

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/MartsinovichDanya/pp_gophermart/internal/accrual"
	"github.com/MartsinovichDanya/pp_gophermart/internal/auth"
	"github.com/MartsinovichDanya/pp_gophermart/internal/config"
	"github.com/MartsinovichDanya/pp_gophermart/internal/handler"
	"github.com/MartsinovichDanya/pp_gophermart/internal/logger"
	"github.com/MartsinovichDanya/pp_gophermart/internal/middleware"
	"github.com/MartsinovichDanya/pp_gophermart/internal/storage"
	"github.com/MartsinovichDanya/pp_gophermart/internal/worker"
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

// Run инициализирует все зависимости, настраивает роутер и запускает HTTP-сервер.
// Поддерживает graceful shutdown при получении сигналов SIGINT или SIGTERM.
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

	// Инициализация клиента системы расчёта баллов
	var accrualClient *accrual.Client
	if cfg.AccrualSystemAddress != "" {
		accrualClient = accrual.NewClient(cfg.AccrualSystemAddress)
		logger.Log.Info("Accrual client initialized", zap.String("address", cfg.AccrualSystemAddress))
	} else {
		logger.Log.Warn("Accrual system address not configured, worker will not start")
	}

	// Инициализация handler
	h := handler.NewHandler(store, cfg.JWTSecret, cfg.MaxBodySize)

	// Инициализация роутера
	r := chi.NewRouter()
	r.Use(chiMiddleware.RequestID)
	r.Use(logger.GetLogger())
	r.Use(middleware.GzipMiddleware)
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

		// Баланс
		r.Get("/api/user/balance", h.GetBalanceHandler)
		r.Post("/api/user/balance/withdraw", h.WithdrawHandler)

		// Выводы
		r.Get("/api/user/withdrawals", h.GetWithdrawalsHandler)
	})

	// Health-check эндпоинт
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Gophermart is running"))
	})

	// Создаём HTTP сервер
	srv := &http.Server{
		Addr:    cfg.RunAddress,
		Handler: r,
	}

	// Запуск воркера обработки заказов (если настроен адрес системы расчёта)
	var orderProcessor *worker.OrderProcessor
	if accrualClient != nil {
		orderProcessor = worker.NewOrderProcessor(store, accrualClient, cfg.ProcessingInterval)
		orderProcessor.Start()
	}

	// Канал для сигналов ОС
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Запуск сервера в горутине
	errCh := make(chan error, 1)
	go func() {
		logger.Log.Info("Starting server", zap.String("address", cfg.RunAddress))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	// Ожидаем сигнал остановки или ошибку сервера
	select {
	case err := <-errCh:
		logger.Log.Error("Server error", zap.Error(err))
		if orderProcessor != nil {
			orderProcessor.Stop()
		}
		return err
	case sig := <-quit:
		logger.Log.Info("Received shutdown signal", zap.String("signal", sig.String()))
	}

	// Graceful shutdown
	logger.Log.Info("Shutting down server...")

	// Создаём контекст с таймаутом для graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Останавливаем воркер
	if orderProcessor != nil {
		orderProcessor.Stop()
	}

	// Останавливаем HTTP сервер
	if err := srv.Shutdown(ctx); err != nil {
		logger.Log.Error("Server shutdown error", zap.Error(err))
		return err
	}

	logger.Log.Info("Server stopped gracefully")
	return nil
}
