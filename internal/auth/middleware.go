package auth

import (
	"context"
	"net/http"
	"strings"

	"go.uber.org/zap"

	"github.com/MartsinovichDanya/pp_gophermart/internal/logger"
)

// contextKey — тип для ключей контекста.
type contextKey string

// UserIDKey — ключ для хранения userID в контексте.
const UserIDKey contextKey = "userID"

// AuthMiddleware оборачивает http.HandlerFunc в проверку JWT токена.
// Токен может быть передан через cookie "token" или заголовок Authorization: Bearer <token>.
// При успешной проверке userID добавляется в контекст запроса.
// При ошибке возвращает 401 Unauthorized.
func AuthMiddleware(tokenSecret string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var tokenString string

		// Пытаемся получить токен из cookie
		cookie, err := r.Cookie("token")
		if err == nil && cookie.Value != "" {
			tokenString = cookie.Value
		} else {
			// Пытаемся получить токен из заголовка Authorization
			authHeader := r.Header.Get("Authorization")
			if authHeader != "" {
				parts := strings.SplitN(authHeader, " ", 2)
				if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
					tokenString = parts[1]
				}
			}
		}

		// Если токен не найден
		if tokenString == "" {
			logger.Log.Debug("Токен не найден в запросе")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Парсим токен
		claims, err := ParseJWT(tokenString, tokenSecret)
		if err != nil {
			logger.Log.Debug("Невалидный токен", zap.Error(err))
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Добавляем userID в контекст
		ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// GetUserIDFromContext извлекает userID из контекста запроса.
func GetUserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(UserIDKey).(string)
	return userID, ok
}
