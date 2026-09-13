package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestGzipMiddleware_CompressJSON проверяет сжатие JSON ответа.
func TestGzipMiddleware_CompressJSON(t *testing.T) {
	handler := GzipMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "hello"}`))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, "gzip", w.Header().Get("Content-Encoding"))

	// Проверяем, что ответ можно распаковать
	gr, err := gzip.NewReader(w.Body)
	assert.NoError(t, err)
	defer gr.Close()

	body, err := io.ReadAll(gr)
	assert.NoError(t, err)
	assert.Equal(t, `{"message": "hello"}`, string(body))
}

// TestGzipMiddleware_NoGzipSupport проверяет ответ без сжатия.
func TestGzipMiddleware_NoGzipSupport(t *testing.T) {
	handler := GzipMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "hello"}`))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// Не устанавливаем Accept-Encoding: gzip
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Empty(t, w.Header().Get("Content-Encoding"))
	assert.Equal(t, `{"message": "hello"}`, w.Body.String())
}

// TestGzipMiddleware_DecompressRequest проверяет распаковку gzip запроса.
func TestGzipMiddleware_DecompressRequest(t *testing.T) {
	var receivedBody string

	handler := GzipMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		receivedBody = string(body)
		w.WriteHeader(http.StatusOK)
	}))

	// Создаём gzip-сжатое тело
	var buf strings.Builder
	gw := gzip.NewWriter(&buf)
	gw.Write([]byte(`{"test": "data"}`))
	gw.Close()

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(buf.String()))
	req.Header.Set("Content-Encoding", "gzip")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, `{"test": "data"}`, receivedBody)
}
