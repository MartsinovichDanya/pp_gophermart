package handler

import (
	"encoding/json"
	"net/http"

	"go.uber.org/zap"

	"github.com/MartsinovichDanya/pp_gophermart/internal/auth"
	"github.com/MartsinovichDanya/pp_gophermart/internal/logger"
	"github.com/MartsinovichDanya/pp_gophermart/internal/model"
)

// GetWithdrawalsHandler обрабатывает GET /api/user/withdrawals.
// Возвращает информацию о выводе средств с накопительного счёта пользователем.
func (h *Handler) GetWithdrawalsHandler(w http.ResponseWriter, r *http.Request) {
	// Получаем userID из контекста
	userID, ok := auth.GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Получаем выводы из хранилища
	withdrawals, err := h.Store.GetUserWithdrawals(r.Context(), userID)
	if err != nil {
		logger.Log.Error("Ошибка получения выводов", zap.Error(err))
		writeJSONError(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}

	// Если выводов нет
	if len(withdrawals) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Преобразуем в формат ответа
	response := make([]model.WithdrawalResponse, 0, len(withdrawals))
	for _, w := range withdrawals {
		resp := model.WithdrawalResponse{
			Order:       w.OrderNumber,
			Sum:         w.Sum,
			ProcessedAt: w.ProcessedAt,
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
