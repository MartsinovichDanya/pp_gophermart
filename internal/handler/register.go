package handler

import (
	"net/http"

	"github.com/mailru/easyjson"
	"go.uber.org/zap"

	"github.com/MartsinovichDanya/pp_gophermart/internal/auth"
	"github.com/MartsinovichDanya/pp_gophermart/internal/logger"
	"github.com/MartsinovichDanya/pp_gophermart/internal/model"
	"github.com/MartsinovichDanya/pp_gophermart/internal/storage"
)

// RegisterHandler обрабатывает POST /api/user/register.
// Регистрирует нового пользователя и автоматически аутентифицирует его.
func (h *Handler) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	// Читаем тело запроса
	body, err := readBody(r, h.MaxBodySize)
	if err != nil {
		logger.Log.Debug("Ошибка чтения тела запроса", zap.Error(err))
		writeJSONError(w, "Ошибка чтения запроса", http.StatusBadRequest)
		return
	}

	// Парсим JSON
	var req model.RegisterRequest
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

	// Хешируем пароль
	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		logger.Log.Error("Ошибка хеширования пароля", zap.Error(err))
		writeJSONError(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}

	// Создаём пользователя
	userID, err := h.Store.CreateUser(r.Context(), req.Login, passwordHash)
	if err != nil {
		if err == storage.ErrUserAlreadyExists {
			logger.Log.Debug("Пользователь уже существует", zap.String("login", req.Login))
			writeJSONError(w, "Логин уже занят", http.StatusConflict)
			return
		}
		logger.Log.Error("Ошибка создания пользователя", zap.Error(err))
		writeJSONError(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}

	logger.Log.Info("Пользователь зарегистрирован", zap.String("userID", userID), zap.String("login", req.Login))

	// Создаём JWT токен
	token, err := auth.NewJWT(userID, h.JWTSecret)
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
