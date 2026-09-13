package handler

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/MartsinovichDanya/pp_gophermart/internal/storage"
)

// Handler содержит зависимости HTTP-обработчиков.
type Handler struct {
	Store       storage.Store
	JWTSecret   string
	MaxBodySize int
}

// NewHandler создаёт новый экземпляр Handler.
func NewHandler(store storage.Store, jwtSecret string, maxBodySize int) *Handler {
	return &Handler{
		Store:       store,
		JWTSecret:   jwtSecret,
		MaxBodySize: maxBodySize,
	}
}

// readBody читает тело запроса с ограничением по размеру.
func readBody(r *http.Request, maxSize int) ([]byte, error) {
	return io.ReadAll(io.LimitReader(r.Body, int64(maxSize)))
}

// writeJSONError отправляет JSON-ошибку с заданным статусом.
func writeJSONError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
