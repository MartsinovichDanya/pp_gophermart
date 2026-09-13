package model

import "time"

//go:generate easyjson -all $GOFILE

// OrderStatus представляет статус обработки заказа.
type OrderStatus string

// Возможные статусы заказа.
const (
	OrderStatusNew        OrderStatus = "NEW"
	OrderStatusProcessing OrderStatus = "PROCESSING"
	OrderStatusInvalid    OrderStatus = "INVALID"
	OrderStatusProcessed  OrderStatus = "PROCESSED"
)

// AccrualStatus представляет статус расчёта начислений во внешней системе.
type AccrualStatus string

// Возможные статусы расчёта начислений.
const (
	AccrualStatusRegistered AccrualStatus = "REGISTERED"
	AccrualStatusProcessing AccrualStatus = "PROCESSING"
	AccrualStatusProcessed  AccrualStatus = "PROCESSED"
	AccrualStatusInvalid    AccrualStatus = "INVALID"
)

// RegisterRequest представляет запрос на регистрацию пользователя.
//
//easyjson:json
type RegisterRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// LoginRequest представляет запрос на аутентификацию пользователя.
//
//easyjson:json
type LoginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// OrderResponse представляет информацию о заказе в списке заказов пользователя.
//
//easyjson:json
type OrderResponse struct {
	Number     string      `json:"number"`
	Status     OrderStatus `json:"status"`
	Accrual    *float64    `json:"accrual,omitempty"`
	UploadedAt time.Time   `json:"uploaded_at"`
}

// BalanceResponse представляет информацию о балансе пользователя.
//
//easyjson:json
type BalanceResponse struct {
	Current   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn"`
}

// WithdrawRequest представляет запрос на списание баллов.
//
//easyjson:json
type WithdrawRequest struct {
	Order string  `json:"order"`
	Sum   float64 `json:"sum"`
}

// WithdrawalResponse представляет информацию о выводе средств.
//
//easyjson:json
type WithdrawalResponse struct {
	Order       string    `json:"order"`
	Sum         float64   `json:"sum"`
	ProcessedAt time.Time `json:"processed_at"`
}

// AccrualResponse представляет ответ от внешней системы расчёта баллов.
//
//easyjson:json
type AccrualResponse struct {
	Order   string        `json:"order"`
	Status  AccrualStatus `json:"status"`
	Accrual *float64      `json:"accrual,omitempty"`
}
