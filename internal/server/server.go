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

	// Роутер на стандартном ServeMux (Go 1.22+)
	mux := http.NewServeMux()

	// Публичные маршруты
	mux.HandleFunc("POST /api/user/register", h.RegisterHandler)
	mux.HandleFunc("POST /api/user/login", h.LoginHandler)

	// Защищённые маршруты
	mux.HandleFunc("POST /api/user/orders", auth.AuthMiddleware(cfg.JWTSecret, h.UploadOrderHandler))
	mux.HandleFunc("GET /api/user/orders", auth.AuthMiddleware(cfg.JWTSecret, h.GetOrdersHandler))
	mux.HandleFunc("GET /api/user/balance", auth.AuthMiddleware(cfg.JWTSecret, h.GetBalanceHandler))
	mux.HandleFunc("POST /api/user/balance/withdraw", auth.AuthMiddleware(cfg.JWTSecret, h.WithdrawHandler))
	mux.HandleFunc("GET /api/user/withdrawals", auth.AuthMiddleware(cfg.JWTSecret, h.GetWithdrawalsHandler))

	// Health-check ({$} — точное совпадение /)
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Gophermart is running"))
	})

	// Цепочка middleware: recover → logger → gzip → mux
	var srvHandler http.Handler = mux
	srvHandler = middleware.GzipMiddleware(srvHandler)
	srvHandler = logger.GetLogger()(srvHandler)
	srvHandler = middleware.RecoverMiddleware(srvHandler)

	srv := &http.Server{
		Addr:              cfg.RunAddress,
		Handler:           srvHandler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	// Запуск воркера
	var orderProcessor *worker.OrderProcessor
	if accrualClient != nil {
		orderProcessor = worker.NewOrderProcessor(store, accrualClient, cfg.ProcessingInterval)
		orderProcessor.Start()
	}

	// Канал для сигналов ОС
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	errCh := make(chan error, 1)
	go func() {
		logger.Log.Info("Starting server", zap.String("address", cfg.RunAddress))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

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
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if orderProcessor != nil {
		orderProcessor.Stop()
	}
	if err := srv.Shutdown(ctx); err != nil {
		logger.Log.Error("Server shutdown error", zap.Error(err))
		return err
	}

	logger.Log.Info("Server stopped gracefully")
	return nil
}
