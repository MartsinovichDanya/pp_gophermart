package handler

import (
	"errors"
	"net/http"

	"github.com/mailru/easyjson"
	"go.uber.org/zap"

	"github.com/MartsinovichDanya/pp_gophermart/internal/auth"
	"github.com/MartsinovichDanya/pp_gophermart/internal/logger"
	"github.com/MartsinovichDanya/pp_gophermart/internal/model"
	"github.com/MartsinovichDanya/pp_gophermart/internal/storage"
)

// GetBalanceHandler обрабатывает GET /api/user/balance.
// Возвращает текущий баланс и сумму выведенных баллов пользователя.
func (h *Handler) GetBalanceHandler(w http.ResponseWriter, r *http.Request) {
	// Получаем userID из контекста
	userID, ok := auth.GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Получаем баланс из хранилища
	current, withdrawn, err := h.Store.GetBalance(r.Context(), userID)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		logger.Log.Error("Ошибка получения баланса", zap.Error(err))
		writeJSONError(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}

	// Формируем ответ
	resp := model.BalanceResponse{
		Current:   current,
		Withdrawn: withdrawn,
	}

	jsonResp, err := easyjson.Marshal(resp)
	if err != nil {
		logger.Log.Error("Ошибка сериализации ответа", zap.Error(err))
		writeJSONError(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(jsonResp)
}

// WithdrawHandler обрабатывает POST /api/user/balance/withdraw.
// Списывает баллы со счёта пользователя в счёт оплаты нового заказа.
func (h *Handler) WithdrawHandler(w http.ResponseWriter, r *http.Request) {
	// Получаем userID из контекста
	userID, ok := auth.GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Читаем тело запроса
	body, err := readBody(r, h.MaxBodySize)
	if err != nil {
		logger.Log.Debug("Ошибка чтения тела запроса", zap.Error(err))
		writeJSONError(w, "Ошибка чтения запроса", http.StatusBadRequest)
		return
	}

	// Парсим JSON
	var req model.WithdrawRequest
	if err := easyjson.Unmarshal(body, &req); err != nil {
		logger.Log.Debug("Ошибка парсинга JSON", zap.Error(err))
		writeJSONError(w, "Некорректный JSON", http.StatusBadRequest)
		return
	}

	// Валидация номера заказа
	if req.Order == "" {
		writeJSONError(w, "Номер заказа не может быть пустым", http.StatusUnprocessableEntity)
		return
	}

	// Проверка, что номер состоит только из цифр
	for _, ch := range req.Order {
		if ch < '0' || ch > '9' {
			writeJSONError(w, "Неверный номер заказа", http.StatusUnprocessableEntity)
			return
		}
	}

	// Валидация суммы
	if req.Sum <= 0 {
		writeJSONError(w, "Сумма должна быть больше нуля", http.StatusBadRequest)
		return
	}

	// Списываем баллы
	err = h.Store.WithdrawBalance(r.Context(), userID, req.Order, req.Sum)
	if err != nil {
		if errors.Is(err, storage.ErrInsufficientBalance) {
			logger.Log.Debug("Недостаточно средств", zap.String("userID", userID), zap.Float64("sum", req.Sum))
			writeJSONError(w, "На счету недостаточно средств", http.StatusPaymentRequired)
			return
		}
		logger.Log.Error("Ошибка списания баллов", zap.Error(err))
		writeJSONError(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}

	logger.Log.Info("Баллы списаны",
		zap.String("userID", userID),
		zap.String("order", req.Order),
		zap.Float64("sum", req.Sum),
	)

	w.WriteHeader(http.StatusOK)
}
