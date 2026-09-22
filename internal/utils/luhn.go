package utils

import (
	"strconv"
	"unicode"
)

// Luhn проверяет корректность номера заказа по алгоритму Луна.
// Номер заказа должен состоять только из цифр и проходить проверку контрольной суммы.
// Возвращает true, если номер валиден, false в противном случае.
func Luhn(number string) bool {
	if number == "" {
		return false
	}

	// Проверяем, что все символы — цифры
	for _, ch := range number {
		if !unicode.IsDigit(ch) {
			return false
		}
	}

	// Алгоритм Луна
	sum := 0
	nDigits := len(number)
	parity := nDigits % 2

	for i := 0; i < nDigits; i++ {
		digit, _ := strconv.Atoi(string(number[i]))

		if i%2 == parity {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}

		sum += digit
	}

	return sum%10 == 0
}
