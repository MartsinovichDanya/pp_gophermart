package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAuthMiddleware_ValidCookie проверяет авторизацию через cookie.
func TestAuthMiddleware_ValidCookie(t *testing.T) {
	secret := "test-secret"
	userID := "user-123"

	token, err := NewJWT(userID, secret)
	require.NoError(t, err)

	handler := AuthMiddleware(secret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := GetUserIDFromContext(r.Context())
		assert.True(t, ok)
		assert.Equal(t, userID, id)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "token", Value: token})
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestAuthMiddleware_ValidHeader проверяет авторизацию через Authorization header.
func TestAuthMiddleware_ValidHeader(t *testing.T) {
	secret := "test-secret"
	userID := "user-123"

	token, err := NewJWT(userID, secret)
	require.NoError(t, err)

	handler := AuthMiddleware(secret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := GetUserIDFromContext(r.Context())
		assert.True(t, ok)
		assert.Equal(t, userID, id)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestAuthMiddleware_NoToken проверяет запрос без токена.
func TestAuthMiddleware_NoToken(t *testing.T) {
	secret := "test-secret"

	handler := AuthMiddleware(secret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestAuthMiddleware_InvalidToken проверяет запрос с невалидным токеном.
func TestAuthMiddleware_InvalidToken(t *testing.T) {
	secret := "test-secret"

	handler := AuthMiddleware(secret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "token", Value: "invalid-token"})
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
