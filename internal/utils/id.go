package utils

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

// base62Chars — алфавит для генерации коротких идентификаторов.
var base62Chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// GenerateID создаёт криптографически случайный идентификатор заданной длины.
// Использует base62 алфавит (a-z, A-Z, 0-9).
// Возвращает ошибку, если не удалось получить случайные данные.
func GenerateID(length int) (string, error) {
	id := make([]byte, length)
	for i := range id {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(base62Chars))))
		if err != nil {
			return "", fmt.Errorf("ошибка генерации случайного числа: %w", err)
		}
		id[i] = base62Chars[idx.Int64()]
	}
	return string(id), nil
}
