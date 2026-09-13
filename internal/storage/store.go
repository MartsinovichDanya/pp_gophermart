package storage

import (
	"context"
	"errors"
	"time"

	"github.com/MartsinovichDanya/pp_gophermart/internal/model"
)

// Предопределённые ошибки хранилища.
var (
	ErrUserAlreadyExists   = errors.New("пользователь уже существует")
	ErrUserNotFound        = errors.New("пользователь не найден")
	ErrOrderAlreadyExists  = errors.New("номер заказа уже загружен другим пользователем")
	ErrOrderOwnedByUser    = errors.New("номер заказа уже загружен этим пользователем")
	ErrInsufficientBalance = errors.New("недостаточно средств на счёте")
)

// User представляет запись пользователя в хранилище.
type User struct {
	ID           string
	Login        string
	PasswordHash string
	Balance      float64
	Withdrawn    float64
}

// Order представляет запись заказа в хранилище.
type Order struct {
	ID         string
	UserID     string
	Number     string
	Status     model.OrderStatus
	Accrual    *float64
	UploadedAt time.Time
}

// Withdrawal представляет запись вывода средств в хранилище.
type Withdrawal struct {
	ID          string
	UserID      string
	OrderNumber string
	Sum         float64
	ProcessedAt time.Time
}

// Store описывает интерфейс хранилища данных.
type Store interface {
	// CreateUser создаёт нового пользователя.
	// Возвращает ErrUserAlreadyExists, если логин уже занят.
	CreateUser(ctx context.Context, login, passwordHash string) (string, error)

	// GetUserByLogin возвращает пользователя по логину.
	// Возвращает ErrUserNotFound, если пользователь не найден.
	GetUserByLogin(ctx context.Context, login string) (*User, error)

	// GetBalance возвращает текущий баланс и сумму выводов пользователя.
	GetBalance(ctx context.Context, userID string) (current, withdrawn float64, err error)

	// CreateOrder создаёт новый заказ для пользователя.
	// Возвращает ErrOrderOwnedByUser, если заказ уже загружен этим пользователем.
	// Возвращает ErrOrderAlreadyExists, если заказ загружен другим пользователем.
	CreateOrder(ctx context.Context, userID, orderNumber string) error

	// GetOrderByNumber возвращает заказ по номеру.
	GetOrderByNumber(ctx context.Context, orderNumber string) (*Order, error)

	// GetUserOrders возвращает все заказы пользователя, отсортированные по убыванию времени загрузки.
	GetUserOrders(ctx context.Context, userID string) ([]Order, error)

	// UpdateOrderStatus обновляет статус и начисление заказа.
	UpdateOrderStatus(ctx context.Context, orderNumber string, status model.OrderStatus, accrual *float64) error

	// GetOrdersToProcess возвращает заказы в статусе NEW или PROCESSING для обработки воркером.
	GetOrdersToProcess(ctx context.Context, limit int) ([]Order, error)

	// AccrueBalance начисляет баллы на счёт пользователя атомарно с обновлением статуса заказа.
	AccrueBalance(ctx context.Context, userID, orderNumber string, accrual float64) error

	// WithdrawBalance списывает баллы со счёта пользователя и создаёт запись о выводе.
	// Возвращает ErrInsufficientBalance, если на счёте недостаточно средств.
	WithdrawBalance(ctx context.Context, userID, orderNumber string, sum float64) error

	// GetUserWithdrawals возвращает все выводы средств пользователя, отсортированные по убыванию времени.
	GetUserWithdrawals(ctx context.Context, userID string) ([]Withdrawal, error)

	// Ping проверяет доступность базы данных.
	Ping(ctx context.Context) error

	// Close закрывает соединение с хранилищем.
	Close()
}
