package worker

import (
	"context"
	"errors"
	"sync"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

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
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
}

// NewOrderProcessor создаёт новый воркер обработки заказов.
// store — хранилище для работы с заказами
// accrualClient — клиент системы расчёта баллов
// interval — интервал между циклами обработки
func NewOrderProcessor(store storage.Store, accrualClient *accrual.Client, interval time.Duration) *OrderProcessor {
	ctx, cancel := context.WithCancel(context.Background())
	return &OrderProcessor{
		store:         store,
		accrualClient: accrualClient,
		interval:      interval,
		ctx:           ctx,
		cancel:        cancel,
	}
}

// Start запускает фоновый воркер в отдельной горутине.
func (op *OrderProcessor) Start() {
	op.wg.Add(1)
	go op.run()
	logger.Log.Info("Воркер обработки заказов запущен", zap.Duration("interval", op.interval))
}

// Stop отменяет контекст воркера и ждёт завершения всех операций.
// Все запросы к БД и внешней системе будут отменены через контекст.
func (op *OrderProcessor) Stop() {
	logger.Log.Info("Остановка воркера обработки заказов...")
	op.cancel()
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
		case <-op.ctx.Done():
			logger.Log.Debug("Получен сигнал остановки воркера")
			return
		case <-ticker.C:
			op.processOrders()
		}
	}
}

// processOrders обрабатывает все заказы в статусах NEW и PROCESSING параллельно.
func (op *OrderProcessor) processOrders() {
	orders, err := op.store.GetOrdersToProcess(op.ctx, 100)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return
		}
		logger.Log.Error("Ошибка получения заказов для обработки", zap.Error(err))
		return
	}

	if len(orders) == 0 {
		logger.Log.Debug("Нет заказов для обработки")
		return
	}

	logger.Log.Info("Начата обработка заказов", zap.Int("count", len(orders)))

	g, gCtx := errgroup.WithContext(op.ctx)
	g.SetLimit(10)

	for _, order := range orders {
		order := order
		g.Go(func() error {
			op.processOrder(gCtx, order)
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		if !errors.Is(err, context.Canceled) {
			logger.Log.Error("Ошибка обработки заказов", zap.Error(err))
		}
	}
}

// processOrder обрабатывает один заказ.
func (op *OrderProcessor) processOrder(ctx context.Context, order storage.Order) {
	logger.Log.Debug("Обработка заказа",
		zap.String("order", order.Number),
		zap.String("status", string(order.Status)),
	)

	response, err := op.accrualClient.CheckOrder(ctx, order.Number)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return
		}
		if errors.Is(err, accrual.ErrOrderNotFound) {
			logger.Log.Debug("Заказ не найден в системе расчёта", zap.String("order", order.Number))
			if err := op.store.UpdateOrderStatus(ctx, order.Number, model.OrderStatusInvalid, nil); err != nil {
				logger.Log.Error("Ошибка обновления статуса заказа", zap.Error(err), zap.String("order", order.Number))
			}
			return
		}
		if errors.Is(err, accrual.ErrRateLimited) {
			logger.Log.Warn("Превышен лимит запросов, пропускаем заказ", zap.String("order", order.Number))
			return
		}
		logger.Log.Error("Ошибка запроса к системе расчёта", zap.Error(err), zap.String("order", order.Number))
		return
	}

	switch response.Status {
	case model.AccrualStatusRegistered, model.AccrualStatusProcessing:
		logger.Log.Debug("Заказ в процессе обработки", zap.String("order", order.Number))
		if err := op.store.UpdateOrderStatus(ctx, order.Number, model.OrderStatusProcessing, nil); err != nil {
			logger.Log.Error("Ошибка обновления статуса заказа", zap.Error(err), zap.String("order", order.Number))
		}

	case model.AccrualStatusProcessed:
		if response.Accrual != nil && *response.Accrual > 0 {
			logger.Log.Info("Начисление баллов",
				zap.String("order", order.Number),
				zap.Float64("accrual", *response.Accrual),
			)
			if err := op.store.AccrueBalance(ctx, order.UserID, order.Number, *response.Accrual); err != nil {
				logger.Log.Error("Ошибка начисления баллов", zap.Error(err), zap.String("order", order.Number))
			}
		} else {
			logger.Log.Debug("Заказ обработан без начислений", zap.String("order", order.Number))
			if err := op.store.UpdateOrderStatus(ctx, order.Number, model.OrderStatusProcessed, nil); err != nil {
				logger.Log.Error("Ошибка обновления статуса заказа", zap.Error(err), zap.String("order", order.Number))
			}
		}

	case model.AccrualStatusInvalid:
		logger.Log.Debug("Заказ не принят к расчёту", zap.String("order", order.Number))
		if err := op.store.UpdateOrderStatus(ctx, order.Number, model.OrderStatusInvalid, nil); err != nil {
			logger.Log.Error("Ошибка обновления статуса заказа", zap.Error(err), zap.String("order", order.Number))
		}

	default:
		logger.Log.Warn("Неизвестный статус ответа", zap.String("status", string(response.Status)))
	}
}
