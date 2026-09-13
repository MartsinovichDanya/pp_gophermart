package accrual

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"go.uber.org/zap"

	"github.com/MartsinovichDanya/pp_gophermart/internal/logger"
	"github.com/MartsinovichDanya/pp_gophermart/internal/model"
)

// Предопределённые ошибки клиента.
var (
	ErrOrderNotFound = errors.New("заказ не найден в системе расчёта")
	ErrRateLimited   = errors.New("превышен лимит запросов")
	ErrInternalError = errors.New("внутренняя ошибка системы расчёта")
)

// Client представляет HTTP клиент для взаимодействия с системой расчёта баллов.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient создаёт новый клиент системы расчёта баллов.
// baseURL — адрес внешней системы (например, "http://localhost:8081").
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// CheckOrder запрашивает информацию о расчёте начислений для конкретного заказа.
// Отправляет GET запрос к внешней системе: /api/orders/{number}
// Возвращает AccrualResponse с информацией о статусе и начислении.
func (c *Client) CheckOrder(ctx context.Context, orderNumber string) (*model.AccrualResponse, error) {
	url := fmt.Sprintf("%s/api/orders/%s", c.baseURL, orderNumber)

	logger.Log.Debug("Запрос к системе расчёта баллов",
		zap.String("url", url),
		zap.String("order", orderNumber),
	)

	// Создаём запрос
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания запроса: %w", err)
	}

	// Выполняем запрос
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ошибка выполнения запроса: %w", err)
	}
	defer resp.Body.Close()

	logger.Log.Debug("Ответ от системы расчёта баллов",
		zap.Int("status", resp.StatusCode),
		zap.String("order", orderNumber),
	)

	// Обрабатываем коды ответов
	switch resp.StatusCode {
	case http.StatusOK:
		// Успешный ответ
		return c.parseResponse(resp.Body, orderNumber)

	case http.StatusNoContent:
		// Заказ не зарегистрирован в системе
		return nil, ErrOrderNotFound

	case http.StatusTooManyRequests:
		// Превышен лимит запросов
		retryAfter := resp.Header.Get("Retry-After")
		logger.Log.Warn("Превышен лимит запросов к системе расчёта",
			zap.String("order", orderNumber),
			zap.String("retryAfter", retryAfter),
		)
		return nil, ErrRateLimited

	case http.StatusInternalServerError:
		// Внутренняя ошибка внешней системы
		logger.Log.Error("Внутренняя ошибка системы расчёта баллов",
			zap.String("order", orderNumber),
		)
		return nil, ErrInternalError

	default:
		// Неожиданный код ответа
		logger.Log.Error("Неожиданный код ответа от системы расчёта",
			zap.Int("status", resp.StatusCode),
			zap.String("order", orderNumber),
		)
		return nil, fmt.Errorf("неожиданный код ответа: %d", resp.StatusCode)
	}
}

// parseResponse парсит JSON ответ от системы расчёта.
func (c *Client) parseResponse(body io.Reader, orderNumber string) (*model.AccrualResponse, error) {
	var response model.AccrualResponse

	if err := json.NewDecoder(body).Decode(&response); err != nil {
		return nil, fmt.Errorf("ошибка парсинга ответа: %w", err)
	}

	logger.Log.Debug("Получен ответ от системы расчёта",
		zap.String("order", orderNumber),
		zap.String("status", string(response.Status)),
		zap.Float64p("accrual", response.Accrual),
	)

	return &response, nil
}

// GetRetryAfterDuration парсит header Retry-After и возвращает duration.
// Поддерживает форматы: секунды (число) и HTTP-дата.
func GetRetryAfterDuration(resp *http.Response) time.Duration {
	retryAfter := resp.Header.Get("Retry-After")
	if retryAfter == "" {
		return 0
	}

	// Пытаемся парсить как число секунд
	if seconds, err := strconv.Atoi(retryAfter); err == nil {
		return time.Duration(seconds) * time.Second
	}

	// Пытаемся парсить как HTTP-дата
	if t, err := http.ParseTime(retryAfter); err == nil {
		return time.Until(t)
	}

	return 0
}
