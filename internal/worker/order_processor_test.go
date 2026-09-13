package worker

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	"github.com/MartsinovichDanya/pp_gophermart/internal/accrual"
	"github.com/MartsinovichDanya/pp_gophermart/internal/model"
	"github.com/MartsinovichDanya/pp_gophermart/internal/storage"
)

// MockStore — мок для storage.Store
type MockStore struct {
	mock.Mock
}

func (m *MockStore) GetOrdersToProcess(ctx context.Context, limit int) ([]storage.Order, error) {
	args := m.Called(ctx, limit)
	return args.Get(0).([]storage.Order), args.Error(1)
}

func (m *MockStore) UpdateOrderStatus(ctx context.Context, orderNumber string, status model.OrderStatus, accrual *float64) error {
	args := m.Called(ctx, orderNumber, status, accrual)
	return args.Error(0)
}

func (m *MockStore) AccrueBalance(ctx context.Context, userID, orderNumber string, accrual float64) error {
	args := m.Called(ctx, userID, orderNumber, accrual)
	return args.Error(0)
}

// Остальные методы мока (заглушки)
func (m *MockStore) CreateUser(ctx context.Context, login, passwordHash string) (string, error) {
	return "", nil
}
func (m *MockStore) GetUserByLogin(ctx context.Context, login string) (*storage.User, error) {
	return nil, nil
}
func (m *MockStore) GetBalance(ctx context.Context, userID string) (float64, float64, error) {
	return 0, 0, nil
}
func (m *MockStore) CreateOrder(ctx context.Context, userID, orderNumber string) error {
	return nil
}
func (m *MockStore) GetOrderByNumber(ctx context.Context, orderNumber string) (*storage.Order, error) {
	return nil, nil
}
func (m *MockStore) GetUserOrders(ctx context.Context, userID string) ([]storage.Order, error) {
	return nil, nil
}
func (m *MockStore) WithdrawBalance(ctx context.Context, userID, orderNumber string, sum float64) error {
	return nil
}
func (m *MockStore) GetUserWithdrawals(ctx context.Context, userID string) ([]storage.Withdrawal, error) {
	return nil, nil
}
func (m *MockStore) Ping(ctx context.Context) error {
	return nil
}
func (m *MockStore) Close() {}

// TestOrderProcessor_StartStop проверяет запуск и остановку воркера.
func TestOrderProcessor_StartStop(t *testing.T) {
	mockStore := new(MockStore)
	mockStore.On("GetOrdersToProcess", mock.Anything, 100).Return([]storage.Order{}, nil)

	client := accrual.NewClient("http://localhost:8081")
	processor := NewOrderProcessor(mockStore, client, 100*time.Millisecond)

	processor.Start()
	time.Sleep(200 * time.Millisecond)
	processor.Stop()

	// Проверяем, что метод вызывался
	mockStore.AssertCalled(t, "GetOrdersToProcess", mock.Anything, 100)
}

// TestOrderProcessor_ProcessOrder_Processed проверяет обработку заказа со статусом PROCESSED.
func TestOrderProcessor_ProcessOrder_Processed(t *testing.T) {
	// Создаём мок HTTP сервер для системы расчёта баллов
	mockAccrualServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Возвращаем успешный ответ с начислением
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"order": "12345678903", "status": "PROCESSED", "accrual": 500}`))
	}))
	defer mockAccrualServer.Close()

	// Создаём клиент с адресом мок сервера
	client := accrual.NewClient(mockAccrualServer.URL)

	// Создаём мок хранилища
	mockStore := new(MockStore)
	orders := []storage.Order{
		{
			ID:     "1",
			UserID: "user1",
			Number: "12345678903",
			Status: model.OrderStatusNew,
		},
	}

	// Настраиваем ожидания
	mockStore.On("GetOrdersToProcess", mock.Anything, 100).Return(orders, nil)
	mockStore.On("AccrueBalance", mock.Anything, "user1", "12345678903", 500.0).Return(nil)

	// Создаём и запускаем воркер
	processor := NewOrderProcessor(mockStore, client, 100*time.Millisecond)
	processor.Start()

	// Ждём, чтобы воркер успел обработать заказ
	time.Sleep(300 * time.Millisecond)

	// Останавливаем воркер
	processor.Stop()

	// Проверяем, что AccrueBalance был вызван с правильными аргументами
	mockStore.AssertCalled(t, "AccrueBalance", mock.Anything, "user1", "12345678903", 500.0)
}

// TestOrderProcessor_ProcessOrder_Processing проверяет обработку заказа со статусом PROCESSING.
func TestOrderProcessor_ProcessOrder_Processing(t *testing.T) {
	// Создаём мок HTTP сервер
	mockAccrualServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"order": "12345678903", "status": "PROCESSING"}`))
	}))
	defer mockAccrualServer.Close()

	client := accrual.NewClient(mockAccrualServer.URL)

	mockStore := new(MockStore)
	orders := []storage.Order{
		{
			ID:     "1",
			UserID: "user1",
			Number: "12345678903",
			Status: model.OrderStatusNew,
		},
	}

	mockStore.On("GetOrdersToProcess", mock.Anything, 100).Return(orders, nil)
	mockStore.On("UpdateOrderStatus", mock.Anything, "12345678903", model.OrderStatusProcessing, (*float64)(nil)).Return(nil)

	processor := NewOrderProcessor(mockStore, client, 100*time.Millisecond)
	processor.Start()
	time.Sleep(300 * time.Millisecond)
	processor.Stop()

	mockStore.AssertCalled(t, "UpdateOrderStatus", mock.Anything, "12345678903", model.OrderStatusProcessing, (*float64)(nil))
}

// TestOrderProcessor_ProcessOrder_Invalid проверяет обработку заказа со статусом INVALID.
func TestOrderProcessor_ProcessOrder_Invalid(t *testing.T) {
	mockAccrualServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"order": "12345678903", "status": "INVALID"}`))
	}))
	defer mockAccrualServer.Close()

	client := accrual.NewClient(mockAccrualServer.URL)

	mockStore := new(MockStore)
	orders := []storage.Order{
		{
			ID:     "1",
			UserID: "user1",
			Number: "12345678903",
			Status: model.OrderStatusNew,
		},
	}

	mockStore.On("GetOrdersToProcess", mock.Anything, 100).Return(orders, nil)
	mockStore.On("UpdateOrderStatus", mock.Anything, "12345678903", model.OrderStatusInvalid, (*float64)(nil)).Return(nil)

	processor := NewOrderProcessor(mockStore, client, 100*time.Millisecond)
	processor.Start()
	time.Sleep(300 * time.Millisecond)
	processor.Stop()

	mockStore.AssertCalled(t, "UpdateOrderStatus", mock.Anything, "12345678903", model.OrderStatusInvalid, (*float64)(nil))
}
