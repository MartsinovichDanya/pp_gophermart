package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/MartsinovichDanya/pp_gophermart/internal/auth"
	"github.com/MartsinovichDanya/pp_gophermart/internal/model"
	"github.com/MartsinovichDanya/pp_gophermart/internal/storage"
)

// MockStore — мок для storage.Store.
type MockStore struct {
	mock.Mock
}

func (m *MockStore) CreateUser(ctx context.Context, login, passwordHash string) (string, error) {
	args := m.Called(ctx, login, passwordHash)
	return args.String(0), args.Error(1)
}

func (m *MockStore) GetUserByLogin(ctx context.Context, login string) (*storage.User, error) {
	args := m.Called(ctx, login)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*storage.User), args.Error(1)
}

func (m *MockStore) GetBalance(ctx context.Context, userID string) (float64, float64, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(float64), args.Get(1).(float64), args.Error(2)
}

func (m *MockStore) CreateOrder(ctx context.Context, userID, orderNumber string) error {
	args := m.Called(ctx, userID, orderNumber)
	return args.Error(0)
}

func (m *MockStore) GetOrderByNumber(ctx context.Context, orderNumber string) (*storage.Order, error) {
	args := m.Called(ctx, orderNumber)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*storage.Order), args.Error(1)
}

func (m *MockStore) GetUserOrders(ctx context.Context, userID string) ([]storage.Order, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]storage.Order), args.Error(1)
}

func (m *MockStore) UpdateOrderStatus(ctx context.Context, orderNumber string, status model.OrderStatus, accrual *float64) error {
	args := m.Called(ctx, orderNumber, status, accrual)
	return args.Error(0)
}

func (m *MockStore) GetOrdersToProcess(ctx context.Context, limit int) ([]storage.Order, error) {
	args := m.Called(ctx, limit)
	return args.Get(0).([]storage.Order), args.Error(1)
}

func (m *MockStore) AccrueBalance(ctx context.Context, userID, orderNumber string, accrual float64) error {
	args := m.Called(ctx, userID, orderNumber, accrual)
	return args.Error(0)
}

func (m *MockStore) WithdrawBalance(ctx context.Context, userID, orderNumber string, sum float64) error {
	args := m.Called(ctx, userID, orderNumber, sum)
	return args.Error(0)
}

func (m *MockStore) GetUserWithdrawals(ctx context.Context, userID string) ([]storage.Withdrawal, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]storage.Withdrawal), args.Error(1)
}

func (m *MockStore) Ping(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockStore) Close() {}

// addAuthContext добавляет userID в контекст запроса.
func addAuthContext(r *http.Request, userID string) *http.Request {
	ctx := context.WithValue(r.Context(), auth.UserIDKey, userID)
	return r.WithContext(ctx)
}

// TestRegisterHandler_Success проверяет успешную регистрацию.
func TestRegisterHandler_Success(t *testing.T) {
	mockStore := new(MockStore)
	mockStore.On("CreateUser", mock.Anything, "testuser", mock.AnythingOfType("string")).
		Return("user-123", nil)

	h := NewHandler(mockStore, "secret", 2048)

	body := bytes.NewBufferString(`{"login":"testuser","password":"testpass"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/user/register", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.RegisterHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	cookies := w.Result().Cookies()
	found := false
	for _, c := range cookies {
		if c.Name == "token" {
			found = true
			assert.NotEmpty(t, c.Value)
		}
	}
	assert.True(t, found, "Cookie 'token' должен быть установлен")
}

// TestRegisterHandler_DuplicateLogin проверяет регистрацию с существующим логином.
func TestRegisterHandler_DuplicateLogin(t *testing.T) {
	mockStore := new(MockStore)
	mockStore.On("CreateUser", mock.Anything, "testuser", mock.AnythingOfType("string")).
		Return("", storage.ErrUserAlreadyExists)

	h := NewHandler(mockStore, "secret", 2048)

	body := bytes.NewBufferString(`{"login":"testuser","password":"testpass"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/user/register", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.RegisterHandler(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

// TestLoginHandler_Success проверяет успешный логин.
func TestLoginHandler_Success(t *testing.T) {
	mockStore := new(MockStore)
	hash, _ := auth.HashPassword("testpass")
	mockStore.On("GetUserByLogin", mock.Anything, "testuser").
		Return(&storage.User{ID: "user-123", Login: "testuser", PasswordHash: hash}, nil)

	h := NewHandler(mockStore, "secret", 2048)

	body := bytes.NewBufferString(`{"login":"testuser","password":"testpass"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/user/login", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.LoginHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// TestLoginHandler_WrongPassword проверяет логин с неверным паролем.
func TestLoginHandler_WrongPassword(t *testing.T) {
	mockStore := new(MockStore)
	hash, _ := auth.HashPassword("testpass")
	mockStore.On("GetUserByLogin", mock.Anything, "testuser").
		Return(&storage.User{ID: "user-123", Login: "testuser", PasswordHash: hash}, nil)

	h := NewHandler(mockStore, "secret", 2048)

	body := bytes.NewBufferString(`{"login":"testuser","password":"wrongpass"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/user/login", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.LoginHandler(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestUploadOrderHandler_Success проверяет успешную загрузку заказа.
func TestUploadOrderHandler_Success(t *testing.T) {
	mockStore := new(MockStore)
	mockStore.On("CreateOrder", mock.Anything, "user-123", "12345678903").Return(nil)

	h := NewHandler(mockStore, "secret", 2048)

	body := bytes.NewBufferString("12345678903")
	req := httptest.NewRequest(http.MethodPost, "/api/user/orders", body)
	req = addAuthContext(req, "user-123")
	w := httptest.NewRecorder()

	h.UploadOrderHandler(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
}

// TestUploadOrderHandler_InvalidLuhn проверяет загрузку с невалидным номером.
func TestUploadOrderHandler_InvalidLuhn(t *testing.T) {
	mockStore := new(MockStore)
	h := NewHandler(mockStore, "secret", 2048)

	body := bytes.NewBufferString("1234567890")
	req := httptest.NewRequest(http.MethodPost, "/api/user/orders", body)
	req = addAuthContext(req, "user-123")
	w := httptest.NewRecorder()

	h.UploadOrderHandler(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

// TestUploadOrderHandler_AlreadyOwned проверяет повторную загрузку своего заказа.
func TestUploadOrderHandler_AlreadyOwned(t *testing.T) {
	mockStore := new(MockStore)
	mockStore.On("CreateOrder", mock.Anything, "user-123", "12345678903").
		Return(storage.ErrOrderOwnedByUser)

	h := NewHandler(mockStore, "secret", 2048)

	body := bytes.NewBufferString("12345678903")
	req := httptest.NewRequest(http.MethodPost, "/api/user/orders", body)
	req = addAuthContext(req, "user-123")
	w := httptest.NewRecorder()

	h.UploadOrderHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// TestUploadOrderHandler_OwnedByOther проверяет загрузку чужого заказа.
func TestUploadOrderHandler_OwnedByOther(t *testing.T) {
	mockStore := new(MockStore)
	mockStore.On("CreateOrder", mock.Anything, "user-123", "12345678903").
		Return(storage.ErrOrderAlreadyExists)

	h := NewHandler(mockStore, "secret", 2048)

	body := bytes.NewBufferString("12345678903")
	req := httptest.NewRequest(http.MethodPost, "/api/user/orders", body)
	req = addAuthContext(req, "user-123")
	w := httptest.NewRecorder()

	h.UploadOrderHandler(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

// TestGetOrdersHandler_Success проверяет получение списка заказов.
func TestGetOrdersHandler_Success(t *testing.T) {
	mockStore := new(MockStore)
	orders := []storage.Order{
		{
			Number:     "12345678903",
			Status:     model.OrderStatusProcessed,
			UploadedAt: time.Now(),
		},
	}
	mockStore.On("GetUserOrders", mock.Anything, "user-123").Return(orders, nil)

	h := NewHandler(mockStore, "secret", 2048)

	req := httptest.NewRequest(http.MethodGet, "/api/user/orders", nil)
	req = addAuthContext(req, "user-123")
	w := httptest.NewRecorder()

	h.GetOrdersHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response []model.OrderResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.Len(t, response, 1)
	assert.Equal(t, "12345678903", response[0].Number)
}

// TestGetOrdersHandler_NoContent проверяет ответ при отсутствии заказов.
func TestGetOrdersHandler_NoContent(t *testing.T) {
	mockStore := new(MockStore)
	mockStore.On("GetUserOrders", mock.Anything, "user-123").Return([]storage.Order{}, nil)

	h := NewHandler(mockStore, "secret", 2048)

	req := httptest.NewRequest(http.MethodGet, "/api/user/orders", nil)
	req = addAuthContext(req, "user-123")
	w := httptest.NewRecorder()

	h.GetOrdersHandler(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

// TestGetBalanceHandler_Success проверяет получение баланса.
func TestGetBalanceHandler_Success(t *testing.T) {
	mockStore := new(MockStore)
	mockStore.On("GetBalance", mock.Anything, "user-123").Return(500.0, 200.0, nil)

	h := NewHandler(mockStore, "secret", 2048)

	req := httptest.NewRequest(http.MethodGet, "/api/user/balance", nil)
	req = addAuthContext(req, "user-123")
	w := httptest.NewRecorder()

	h.GetBalanceHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp model.BalanceResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, 500.0, resp.Current)
	assert.Equal(t, 200.0, resp.Withdrawn)
}

// TestWithdrawHandler_Success проверяет успешное списание.
func TestWithdrawHandler_Success(t *testing.T) {
	mockStore := new(MockStore)
	mockStore.On("WithdrawBalance", mock.Anything, "user-123", "2377225624", 100.0).Return(nil)

	h := NewHandler(mockStore, "secret", 2048)

	body := bytes.NewBufferString(`{"order":"2377225624","sum":100}`)
	req := httptest.NewRequest(http.MethodPost, "/api/user/balance/withdraw", body)
	req.Header.Set("Content-Type", "application/json")
	req = addAuthContext(req, "user-123")
	w := httptest.NewRecorder()

	h.WithdrawHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// TestWithdrawHandler_InsufficientBalance проверяет списание при недостатке средств.
func TestWithdrawHandler_InsufficientBalance(t *testing.T) {
	mockStore := new(MockStore)
	mockStore.On("WithdrawBalance", mock.Anything, "user-123", "2377225624", 1000.0).
		Return(storage.ErrInsufficientBalance)

	h := NewHandler(mockStore, "secret", 2048)

	body := bytes.NewBufferString(`{"order":"2377225624","sum":1000}`)
	req := httptest.NewRequest(http.MethodPost, "/api/user/balance/withdraw", body)
	req.Header.Set("Content-Type", "application/json")
	req = addAuthContext(req, "user-123")
	w := httptest.NewRecorder()

	h.WithdrawHandler(w, req)

	assert.Equal(t, http.StatusPaymentRequired, w.Code)
}

// TestGetWithdrawalsHandler_Success проверяет получение истории выводов.
func TestGetWithdrawalsHandler_Success(t *testing.T) {
	mockStore := new(MockStore)
	withdrawals := []storage.Withdrawal{
		{
			OrderNumber: "2377225624",
			Sum:         200.0,
			ProcessedAt: time.Now(),
		},
	}
	mockStore.On("GetUserWithdrawals", mock.Anything, "user-123").Return(withdrawals, nil)

	h := NewHandler(mockStore, "secret", 2048)

	req := httptest.NewRequest(http.MethodGet, "/api/user/withdrawals", nil)
	req = addAuthContext(req, "user-123")
	w := httptest.NewRecorder()

	h.GetWithdrawalsHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response []model.WithdrawalResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.Len(t, response, 1)
	assert.Equal(t, "2377225624", response[0].Order)
	assert.Equal(t, 200.0, response[0].Sum)
}

// TestGetWithdrawalsHandler_NoContent проверяет ответ при отсутствии выводов.
func TestGetWithdrawalsHandler_NoContent(t *testing.T) {
	mockStore := new(MockStore)
	mockStore.On("GetUserWithdrawals", mock.Anything, "user-123").Return([]storage.Withdrawal{}, nil)

	h := NewHandler(mockStore, "secret", 2048)

	req := httptest.NewRequest(http.MethodGet, "/api/user/withdrawals", nil)
	req = addAuthContext(req, "user-123")
	w := httptest.NewRecorder()

	h.GetWithdrawalsHandler(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}
