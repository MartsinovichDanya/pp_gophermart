package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHashPassword проверяет хеширование пароля.
func TestHashPassword(t *testing.T) {
	password := "testpassword123"
	hash, err := HashPassword(password)

	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, password, hash)
}

// TestCheckPasswordHash проверяет сравнение пароля с хешем.
func TestCheckPasswordHash(t *testing.T) {
	password := "testpassword123"
	hash, err := HashPassword(password)
	require.NoError(t, err)

	assert.True(t, CheckPasswordHash(password, hash))
	assert.False(t, CheckPasswordHash("wrongpassword", hash))
}

// TestNewJWT проверяет создание JWT токена.
func TestNewJWT(t *testing.T) {
	userID := "user-123"
	secret := "test-secret"

	token, err := NewJWT(userID, secret)
	require.NoError(t, err)
	assert.NotEmpty(t, token)
}

// TestParseJWT проверяет парсинг JWT токена.
func TestParseJWT(t *testing.T) {
	userID := "user-123"
	secret := "test-secret"

	token, err := NewJWT(userID, secret)
	require.NoError(t, err)

	claims, err := ParseJWT(token, secret)
	require.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
}

// TestParseJWT_InvalidToken проверяет обработку невалидного токена.
func TestParseJWT_InvalidToken(t *testing.T) {
	secret := "test-secret"

	_, err := ParseJWT("invalid-token", secret)
	assert.ErrorIs(t, err, ErrInvalidToken)
}

// TestParseJWT_WrongSecret проверяет обработку токена с неверным секретом.
func TestParseJWT_WrongSecret(t *testing.T) {
	userID := "user-123"
	token, err := NewJWT(userID, "correct-secret")
	require.NoError(t, err)

	_, err = ParseJWT(token, "wrong-secret")
	assert.ErrorIs(t, err, ErrInvalidToken)
}
