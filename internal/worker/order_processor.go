package worker

import (
	"context"
	"errors"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/MartsinovichDanya/pp_gophermart/internal/accrual"
	"github.com/MartsinovichDanya/pp_gophermart/internal/logger"
	"github.com/MartsinovichDanya/pp_gophermart/internal/model"
	"github.com/MartsinovichDanya/pp_gophermart/internal/storage"
)

// OrderProcessor представляет фоновый воркер для обработки заказов.
// Периодически проверяет заказы в статусах NEW и PROCESSING,
// отправляет запросы во внешнюю систему расчёта баллов и обновляет статусы.
type OrderProcessor struct {
	store         storage.Store
	accrualClient *accrual.Client
	interval      time.Duration
	stopCh        chan struct{}
	wg            sync.WaitGroup
}

// NewOrderProcessor создаёт новый воркер обработки заказов.
// store — хранилище для работы с заказами
// accrualClient — клиент системы расчёта баллов
// interval — интервал между циклами обработки (рекомендуется 5-10 секунд)
func NewOrderProcessor(store storage.Store, accrualClient *accrual.Client, interval time.Duration) *OrderProcessor {
	return &OrderProcessor{
		store:         store,
		accrualClient: accrualClient,
		interval:      interval,
		stopCh:        make(chan struct{}),
	}
}

// Start запускает фоновый воркер в отдельной горутине.
func (op *OrderProcessor) Start() {
	op.wg.Add(1)
	go op.run()
	logger.Log.Info("Воркер обработки заказов запущен", zap.Duration("interval", op.interval))
}

// Stop корректно останавливает воркер (graceful shutdown).
// Ждёт завершения текущего цикла обработки.
func (op *OrderProcessor) Stop() {
	logger.Log.Info("Остановка воркера обработки заказов...")
	close(op.stopCh)
	op.wg.Wait()
	logger.Log.Info("Воркер обработки заказов остановлен")
}

// run — основной цикл воркера.
func (op *OrderProcessor) run() {
	defer op.wg.Done()

	ticker := time.NewTicker(op.interval)
	defer ticker.Stop()

	for {
		select {
		case <-op.stopCh:
			logger.Log.Debug("Получен сигнал остановки воркера")
			return
		case <-ticker.C:
			op.processOrders()
		}
	}
}

// processOrders обрабатывает все заказы в статусах NEW и PROCESSING.
func (op *OrderProcessor) processOrders() {
	ctx := context.Background()

	// Получаем заказы для обработки (лимит 100 за раз)
	orders, err := op.store.GetOrdersToProcess(ctx, 100)
	if err != nil {
		logger.Log.Error("Ошибка получения заказов для обработки", zap.Error(err))
		return
	}

	if len(orders) == 0 {
		logger.Log.Debug("Нет заказов для обработки")
		return
	}

	logger.Log.Info("Начата обработка заказов", zap.Int("count", len(orders)))

	// Обрабатываем каждый заказ
	for _, order := range orders {
		// Проверяем, не нужно ли остановиться
		select {
		case <-op.stopCh:
			logger.Log.Debug("Остановка воркера во время обработки заказов")
			return
		default:
		}

		op.processOrder(ctx, order)

		// Небольшая пауза между запросами, чтобы не перегружать внешнюю систему
		time.Sleep(100 * time.Millisecond)
	}
}

// processOrder обрабатывает один заказ.
func (op *OrderProcessor) processOrder(ctx context.Context, order storage.Order) {
	logger.Log.Debug("Обработка заказа",
		zap.String("order", order.Number),
		zap.String("status", string(order.Status)),
	)

	// Запрашиваем информацию у системы расчёта баллов
	response, err := op.accrualClient.CheckOrder(ctx, order.Number)
	if err != nil {
		if errors.Is(err, accrual.ErrOrderNotFound) {
			// Заказ не найден во внешней системе — помечаем как INVALID
			logger.Log.Debug("Заказ не найден в системе расчёта", zap.String("order", order.Number))
			if err := op.store.UpdateOrderStatus(ctx, order.Number, model.OrderStatusInvalid, nil); err != nil {
				logger.Log.Error("Ошибка обновления статуса заказа", zap.Error(err), zap.String("order", order.Number))
			}
			return
		}

		if errors.Is(err, accrual.ErrRateLimited) {
			// Превышен лимит запросов — пропускаем и попробуем в следующем цикле
			logger.Log.Warn("Превышен лимит запросов, пропускаем заказ", zap.String("order", order.Number))
			return
		}

		// Другие ошибки — логируем и пропускаем
		logger.Log.Error("Ошибка запроса к системе расчёта", zap.Error(err), zap.String("order", order.Number))
		return
	}

	// Обрабатываем ответ в зависимости от статуса
	switch response.Status {
	case model.AccrualStatusRegistered, model.AccrualStatusProcessing:
		// Заказ ещё обрабатывается — обновляем статус на PROCESSING
		logger.Log.Debug("Заказ в процессе обработки", zap.String("order", order.Number))
		if err := op.store.UpdateOrderStatus(ctx, order.Number, model.OrderStatusProcessing, nil); err != nil {
			logger.Log.Error("Ошибка обновления статуса заказа", zap.Error(err), zap.String("order", order.Number))
		}

	case model.AccrualStatusProcessed:
		// Расчёт завершён — начисляем баллы
		if response.Accrual != nil && *response.Accrual > 0 {
			logger.Log.Info("Начисление баллов",
				zap.String("order", order.Number),
				zap.Float64("accrual", *response.Accrual),
			)
			if err := op.store.AccrueBalance(ctx, order.UserID, order.Number, *response.Accrual); err != nil {
				logger.Log.Error("Ошибка начисления баллов", zap.Error(err), zap.String("order", order.Number))
			}
		} else {
			// Расчёт завершён, но начислений нет — помечаем как PROCESSED без баллов
			logger.Log.Debug("Заказ обработан без начислений", zap.String("order", order.Number))
			if err := op.store.UpdateOrderStatus(ctx, order.Number, model.OrderStatusProcessed, nil); err != nil {
				logger.Log.Error("Ошибка обновления статуса заказа", zap.Error(err), zap.String("order", order.Number))
			}
		}

	case model.AccrualStatusInvalid:
		// Заказ не принят к расчёту — помечаем как INVALID
		logger.Log.Debug("Заказ не принят к расчёту", zap.String("order", order.Number))
		if err := op.store.UpdateOrderStatus(ctx, order.Number, model.OrderStatusInvalid, nil); err != nil {
			logger.Log.Error("Ошибка обновления статуса заказа", zap.Error(err), zap.String("order", order.Number))
		}

	default:
		logger.Log.Warn("Неизвестный статус ответа", zap.String("status", string(response.Status)))
	}
}
