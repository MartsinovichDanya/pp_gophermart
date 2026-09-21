package middleware

import (
	"net/http"

	"go.uber.org/zap"

	"github.com/MartsinovichDanya/pp_gophermart/internal/logger"
)

// RecoverMiddleware перехватывает паники в обработчиках и возвращает 500.
func RecoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				logger.Log.Error("panic recovered",
					zap.Any("panic", err),
					zap.String("method", r.Method),
					zap.String("path", r.URL.Path),
				)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
