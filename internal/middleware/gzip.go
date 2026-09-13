package middleware

import (
	"compress/gzip"
	"net/http"
	"strings"

	"go.uber.org/zap"

	"github.com/MartsinovichDanya/pp_gophermart/internal/logger"
)

// GzipMiddleware обрабатывает сжатие входящих запросов и исходящих ответов.
// Распаковывает входящие gzip-запросы и сжимает исходящие ответы для JSON и HTML.
func GzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. Распаковка входящего запроса, если он сжат gzip
		if r.Header.Get("Content-Encoding") == "gzip" {
			origBody := r.Body
			gzr, err := gzip.NewReader(origBody)
			if err != nil {
				logger.Log.Debug("Ошибка распаковки gzip запроса", zap.Error(err))
				http.Error(w, "Invalid gzip body", http.StatusBadRequest)
				return
			}
			r.Body = gzr
			defer origBody.Close()
			r.Header.Del("Content-Encoding")
		}

		// 2. Сжатие исходящего ответа, если клиент поддерживает gzip
		acceptEncoding := r.Header.Get("Accept-Encoding")
		supportsGzip := strings.Contains(acceptEncoding, "gzip")
		gzw := &gzipResponseWriter{
			ResponseWriter: w,
			supportsGzip:   supportsGzip,
		}
		defer gzw.Close()

		next.ServeHTTP(gzw, r)
	})
}

// gzipResponseWriter оборачивает http.ResponseWriter и сжимает данные при необходимости.
type gzipResponseWriter struct {
	http.ResponseWriter
	supportsGzip bool
	writer       *gzip.Writer
	started      bool
}

// WriteHeader устанавливает заголовки и при необходимости создаёт gzip writer.
func (g *gzipResponseWriter) WriteHeader(code int) {
	if g.started {
		return
	}
	g.started = true

	contentType := g.Header().Get("Content-Type")
	// Сжимаем только JSON и HTML
	if g.supportsGzip && (strings.HasPrefix(contentType, "application/json") || strings.HasPrefix(contentType, "text/html")) {
		g.Header().Set("Content-Encoding", "gzip")
		g.Header().Del("Content-Length") // длина будет неизвестна до конца записи
		g.writer = gzip.NewWriter(g.ResponseWriter)
	}
	g.ResponseWriter.WriteHeader(code)
}

// Write записывает данные, сжимая их если создан gzip writer.
func (g *gzipResponseWriter) Write(b []byte) (int, error) {
	// Если Write вызван до WriteHeader, устанавливаем StatusOK
	if !g.started {
		if g.Header().Get("Content-Type") == "" {
			g.Header().Set("Content-Type", http.DetectContentType(b))
		}
		g.WriteHeader(http.StatusOK)
	}

	if g.writer != nil {
		return g.writer.Write(b)
	}
	return g.ResponseWriter.Write(b)
}

// Close закрывает gzip writer и сбрасывает остаток данных.
func (g *gzipResponseWriter) Close() error {
	if g.writer != nil {
		return g.writer.Close()
	}
	return nil
}
