package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// Claims представляет полезную нагрузку JWT токена.
type Claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

// ErrInvalidToken возвращается при невалидном токене.
var ErrInvalidToken = errors.New("невалидный токен")

// HashPassword хеширует пароль с помощью bcrypt.
// Возвращает хеш пароля или ошибку при неудаче.
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("ошибка хеширования пароля: %w", err)
	}
	return string(bytes), nil
}

// CheckPasswordHash сравнивает пароль с хешем.
// Возвращает true, если пароль совпадает с хешем.
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// NewJWT создаёт JWT токен для пользователя.
// Принимает userID и секретный ключ.
// Возвращает строку токена или ошибку при неудаче.
func NewJWT(userID, tokenSecret string) (string, error) {
	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(tokenSecret))
	if err != nil {
		return "", fmt.Errorf("ошибка подписи токена: %w", err)
	}

	return tokenString, nil
}

// ParseJWT парсит и валидирует JWT токен.
// Принимает строку токена и секретный ключ.
// Возвращает Claims или ошибку при невалидном токене.
func ParseJWT(tokenString, tokenSecret string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("неожиданный метод подписи: %v", token.Header["alg"])
		}
		return []byte(tokenSecret), nil
	})

	if err != nil {
		return nil, ErrInvalidToken
	}

	if !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
