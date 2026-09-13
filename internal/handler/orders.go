package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"go.uber.org/zap"

	"github.com/MartsinovichDanya/pp_gophermart/internal/auth"
	"github.com/MartsinovichDanya/pp_gophermart/internal/logger"
	"github.com/MartsinovichDanya/pp_gophermart/internal/model"
	"github.com/MartsinovichDanya/pp_gophermart/internal/storage"
	"github.com/MartsinovichDanya/pp_gophermart/internal/utils"
)

// UploadOrderHandler обрабатывает POST /api/user/orders.
// Загружает номер заказа для расчёта баллов лояльности.
func (h *Handler) UploadOrderHandler(w http.ResponseWriter, r *http.Request) {
	// Получаем userID из контекста
	userID, ok := auth.GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Читаем тело запроса (текст/plain)
	body, err := io.ReadAll(io.LimitReader(r.Body, int64(h.MaxBodySize)))
	if err != nil {
		logger.Log.Debug("Ошибка чтения тела запроса", zap.Error(err))
		writeJSONError(w, "Ошибка чтения запроса", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	orderNumber := strings.TrimSpace(string(body))

	// Валидация формата
	if orderNumber == "" {
		writeJSONError(w, "Номер заказа не может быть пустым", http.StatusBadRequest)
		return
	}

	// Проверка, что номер состоит только из цифр
	for _, ch := range orderNumber {
		if ch < '0' || ch > '9' {
			writeJSONError(w, "Неверный формат номера заказа", http.StatusUnprocessableEntity)
			return
		}
	}

	// Проверка по алгоритму Луна
	if !utils.Luhn(orderNumber) {
		writeJSONError(w, "Неверный формат номера заказа", http.StatusUnprocessableEntity)
		return
	}

	// Создаём заказ
	err = h.Store.CreateOrder(r.Context(), userID, orderNumber)
	if err != nil {
		if errors.Is(err, storage.ErrOrderOwnedByUser) {
			logger.Log.Debug("Заказ уже загружен этим пользователем", zap.String("order", orderNumber))
			w.WriteHeader(http.StatusOK)
			return
		}
		if errors.Is(err, storage.ErrOrderAlreadyExists) {
			logger.Log.Debug("Заказ уже загружен другим пользователем", zap.String("order", orderNumber))
			writeJSONError(w, "Номер заказа уже загружен другим пользователем", http.StatusConflict)
			return
		}
		logger.Log.Error("Ошибка создания заказа", zap.Error(err))
		writeJSONError(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}

	logger.Log.Info("Заказ загружен", zap.String("userID", userID), zap.String("order", orderNumber))
	w.WriteHeader(http.StatusAccepted)
}

// GetOrdersHandler обрабатывает GET /api/user/orders.
// Возвращает список загруженных пользователем номеров заказов.
func (h *Handler) GetOrdersHandler(w http.ResponseWriter, r *http.Request) {
	// Получаем userID из контекста
	userID, ok := auth.GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Получаем заказы из хранилища
	orders, err := h.Store.GetUserOrders(r.Context(), userID)
	if err != nil {
		logger.Log.Error("Ошибка получения заказов", zap.Error(err))
		writeJSONError(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}

	// Если заказов нет
	if len(orders) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Преобразуем в формат ответа
	response := make([]model.OrderResponse, 0, len(orders))
	for _, order := range orders {
		resp := model.OrderResponse{
			Number:     order.Number,
			Status:     order.Status,
			Accrual:    order.Accrual,
			UploadedAt: order.UploadedAt,
		}
		response = append(response, resp)
	}

	// Сериализуем в JSON (используем encoding/json для срезов)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.Log.Error("Ошибка сериализации ответа", zap.Error(err))
		return
	}
}
