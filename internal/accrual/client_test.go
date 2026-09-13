package accrual

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MartsinovichDanya/pp_gophermart/internal/model"
)

// TestCheckOrder_Success проверяет успешный запрос.
func TestCheckOrder_Success(t *testing.T) {
	// Создаём тестовый сервер
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/orders/12345678903", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"order": "12345678903", "status": "PROCESSED", "accrual": 500}`))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	resp, err := client.CheckOrder(context.Background(), "12345678903")

	require.NoError(t, err)
	assert.Equal(t, "12345678903", resp.Order)
	assert.Equal(t, model.AccrualStatusProcessed, resp.Status)
	require.NotNil(t, resp.Accrual)
	assert.Equal(t, float64(500), *resp.Accrual)
}

// TestCheckOrder_NotFound проверяет ответ 204.
func TestCheckOrder_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	resp, err := client.CheckOrder(context.Background(), "99999999999")

	assert.ErrorIs(t, err, ErrOrderNotFound)
	assert.Nil(t, resp)
}

// TestCheckOrder_RateLimited проверяет ответ 429.
func TestCheckOrder_RateLimited(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	resp, err := client.CheckOrder(context.Background(), "12345678903")

	assert.ErrorIs(t, err, ErrRateLimited)
	assert.Nil(t, resp)
}

// TestCheckOrder_Processing проверяет статус PROCESSING.
func TestCheckOrder_Processing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"order": "12345678903", "status": "PROCESSING"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	resp, err := client.CheckOrder(context.Background(), "12345678903")

	require.NoError(t, err)
	assert.Equal(t, model.AccrualStatusProcessing, resp.Status)
	assert.Nil(t, resp.Accrual)
}

// TestCheckOrder_Invalid проверяет статус INVALID.
func TestCheckOrder_Invalid(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"order": "12345678903", "status": "INVALID"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	resp, err := client.CheckOrder(context.Background(), "12345678903")

	require.NoError(t, err)
	assert.Equal(t, model.AccrualStatusInvalid, resp.Status)
	assert.Nil(t, resp.Accrual)
}

// TestGetRetryAfterDuration проверяет парсинг header Retry-After.
func TestGetRetryAfterDuration(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		expected int
	}{
		{"60 seconds", "60", 60},
		{"30 seconds", "30", 30},
		{"empty", "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{
				Header: http.Header{},
			}
			if tt.header != "" {
				resp.Header.Set("Retry-After", tt.header)
			}

			duration := GetRetryAfterDuration(resp)
			assert.Equal(t, tt.expected, int(duration.Seconds()))
		})
	}
}
