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

// LoginHandler обрабатывает POST /api/user/login.
// Аутентифицирует пользователя по логину и паролю.
func (h *Handler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	// Читаем тело запроса
	body, err := readBody(r, h.MaxBodySize)
	if err != nil {
		logger.Log.Debug("Ошибка чтения тела запроса", zap.Error(err))
		writeJSONError(w, "Ошибка чтения запроса", http.StatusBadRequest)
		return
	}

	// Парсим JSON
	var req model.LoginRequest
	if err := easyjson.Unmarshal(body, &req); err != nil {
		logger.Log.Debug("Ошибка парсинга JSON", zap.Error(err))
		writeJSONError(w, "Некорректный JSON", http.StatusBadRequest)
		return
	}

	// Валидация
	if req.Login == "" || req.Password == "" {
		writeJSONError(w, "Логин и пароль обязательны", http.StatusBadRequest)
		return
	}

	// Получаем пользователя
	user, err := h.Store.GetUserByLogin(r.Context(), req.Login)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			logger.Log.Debug("Пользователь не найден", zap.String("login", req.Login))
			writeJSONError(w, "Неверная пара логин/пароль", http.StatusUnauthorized)
			return
		}
		logger.Log.Error("Ошибка получения пользователя", zap.Error(err))
		writeJSONError(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}

	// Проверяем пароль
	if !auth.CheckPasswordHash(req.Password, user.PasswordHash) {
		logger.Log.Debug("Неверный пароль", zap.String("login", req.Login))
		writeJSONError(w, "Неверная пара логин/пароль", http.StatusUnauthorized)
		return
	}

	logger.Log.Info("Пользователь аутентифицирован", zap.String("userID", user.ID), zap.String("login", req.Login))

	// Создаём JWT токен
	token, err := auth.NewJWT(user.ID, h.JWTSecret)
	if err != nil {
		logger.Log.Error("Ошибка создания токена", zap.Error(err))
		writeJSONError(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}

	// Устанавливаем cookie с токеном
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	w.WriteHeader(http.StatusOK)
}
