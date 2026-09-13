package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestLuhn_ValidNumbers проверяет валидные номера заказов.
func TestLuhn_ValidNumbers(t *testing.T) {
	validNumbers := []string{
		"12345678903",
		"9278923470",
		"2377225624",
		"79927398713", // классический пример валидного номера Луна
	}

	for _, number := range validNumbers {
		t.Run(number, func(t *testing.T) {
			assert.True(t, Luhn(number), "Номер %s должен быть валидным", number)
		})
	}
}

// TestLuhn_InvalidNumbers проверяет невалидные номера заказов.
func TestLuhn_InvalidNumbers(t *testing.T) {
	invalidNumbers := []string{
		"1234567890",
		"12345678901",
		"12345678904",
		"12345678901234567890", // не проходит алгоритм Луна
	}

	for _, number := range invalidNumbers {
		t.Run(number, func(t *testing.T) {
			assert.False(t, Luhn(number), "Номер %s должен быть невалидным", number)
		})
	}
}

// TestLuhn_EmptyString проверяет пустую строку.
func TestLuhn_EmptyString(t *testing.T) {
	assert.False(t, Luhn(""), "Пустая строка должна быть невалидной")
}

// TestLuhn_NonDigitCharacters проверяет строки с нецифровыми символами.
func TestLuhn_NonDigitCharacters(t *testing.T) {
	invalidStrings := []string{
		"12345abc",
		"123-456",
		"123 456",
		"abc123",
	}

	for _, str := range invalidStrings {
		t.Run(str, func(t *testing.T) {
			assert.False(t, Luhn(str), "Строка %s должна быть невалидной", str)
		})
	}
}
