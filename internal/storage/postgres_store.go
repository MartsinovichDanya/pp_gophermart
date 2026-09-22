package storage

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/MartsinovichDanya/pp_gophermart/internal/logger"
	"github.com/MartsinovichDanya/pp_gophermart/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// PostgresStore реализует интерфейс Store для PostgreSQL.
type PostgresStore struct {
	pool *pgxpool.Pool
}

// NewPostgresStore создаёт новое хранилище PostgreSQL, применяет миграции.
func NewPostgresStore(databaseURL string) (*PostgresStore, error) {
	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		return nil, fmt.Errorf("не удалось создать пул подключений: %w", err)
	}

	if err := runMigrations(databaseURL); err != nil {
		pool.Close()
		return nil, fmt.Errorf("не удалось применить миграции: %w", err)
	}

	logger.Log.Info("Database migrations applied successfully")
	return &PostgresStore{pool: pool}, nil
}

// runMigrations применяет SQL миграции из embed.FS.
func runMigrations(databaseURL string) error {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return fmt.Errorf("sql.Open: %w", err)
	}
	defer db.Close()

	goose.SetBaseFS(migrationsFS)
	if err := goose.Up(db, "migrations"); err != nil {
		return fmt.Errorf("goose.Up: %w", err)
	}
	return nil
}

// CreateUser создаёт нового пользователя.
func (s *PostgresStore) CreateUser(ctx context.Context, login, passwordHash string) (string, error) {
	var userID string
	err := s.pool.QueryRow(ctx,
		`INSERT INTO service_data.users (login, password_hash) VALUES ($1, $2) RETURNING id`,
		login, passwordHash,
	).Scan(&userID)

	if err != nil {
		if isUniqueViolation(err) {
			return "", ErrUserAlreadyExists
		}
		return "", fmt.Errorf("ошибка создания пользователя: %w", err)
	}
	return userID, nil
}

// GetUserByLogin возвращает пользователя по логину.
func (s *PostgresStore) GetUserByLogin(ctx context.Context, login string) (*User, error) {
	var user User
	err := s.pool.QueryRow(ctx,
		`SELECT id, login, password_hash, balance, withdrawn FROM service_data.users WHERE login = $1`,
		login,
	).Scan(&user.ID, &user.Login, &user.PasswordHash, &user.Balance, &user.Withdrawn)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("ошибка получения пользователя: %w", err)
	}
	return &user, nil
}

// GetBalance возвращает текущий баланс и сумму выводов пользователя.
func (s *PostgresStore) GetBalance(ctx context.Context, userID string) (float64, float64, error) {
	var current, withdrawn float64
	err := s.pool.QueryRow(ctx,
		`SELECT balance, withdrawn FROM service_data.users WHERE id = $1`,
		userID,
	).Scan(&current, &withdrawn)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, 0, ErrUserNotFound
		}
		return 0, 0, fmt.Errorf("ошибка получения баланса: %w", err)
	}
	return current, withdrawn, nil
}

// CreateOrder создаёт новый заказ для пользователя.
func (s *PostgresStore) CreateOrder(ctx context.Context, userID, orderNumber string) error {
	// Атомарная вставка: если номер уже есть — ничего не делаем
	tag, err := s.pool.Exec(ctx,
		`INSERT INTO service_data.orders (user_id, number, status)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (number) DO NOTHING`,
		userID, orderNumber, model.OrderStatusNew,
	)
	if err != nil {
		return fmt.Errorf("ошибка создания заказа: %w", err)
	}

	// RowsAffected == 1 — заказ успешно создан
	if tag.RowsAffected() == 1 {
		return nil
	}

	// RowsAffected == 0 — конфликт, номер уже существует. Определяем владельца.
	var existingUserID string
	err = s.pool.QueryRow(ctx,
		`SELECT user_id FROM service_data.orders WHERE number = $1`,
		orderNumber,
	).Scan(&existingUserID)

	if err != nil {
		return ErrOrderAlreadyExists
	}

	if existingUserID == userID {
		return ErrOrderOwnedByUser
	}
	return ErrOrderAlreadyExists
}

// GetOrderByNumber возвращает заказ по номеру.
func (s *PostgresStore) GetOrderByNumber(ctx context.Context, orderNumber string) (*Order, error) {
	var order Order
	err := s.pool.QueryRow(ctx,
		`SELECT id, user_id, number, status, accrual, uploaded_at FROM service_data.orders WHERE number = $1`,
		orderNumber,
	).Scan(&order.ID, &order.UserID, &order.Number, &order.Status, &order.Accrual, &order.UploadedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("заказ %s не найден: %w", orderNumber, err)
		}
		return nil, fmt.Errorf("ошибка получения заказа: %w", err)
	}
	return &order, nil
}

// GetUserOrders возвращает все заказы пользователя.
func (s *PostgresStore) GetUserOrders(ctx context.Context, userID string) ([]Order, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, user_id, number, status, accrual, uploaded_at 
		 FROM service_data.orders WHERE user_id = $1 
		 ORDER BY uploaded_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения заказов: %w", err)
	}
	defer rows.Close()

	var orders []Order
	for rows.Next() {
		var order Order
		if err := rows.Scan(&order.ID, &order.UserID, &order.Number, &order.Status, &order.Accrual, &order.UploadedAt); err != nil {
			return nil, fmt.Errorf("ошибка чтения строки: %w", err)
		}
		orders = append(orders, order)
	}
	return orders, rows.Err()
}

// UpdateOrderStatus обновляет статус и начисление заказа.
func (s *PostgresStore) UpdateOrderStatus(ctx context.Context, orderNumber string, status model.OrderStatus, accrual *float64) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE service_data.orders SET status = $1, accrual = $2 WHERE number = $3`,
		status, accrual, orderNumber,
	)
	if err != nil {
		return fmt.Errorf("ошибка обновления статуса: %w", err)
	}
	return nil
}

// GetOrdersToProcess возвращает заказы для обработки воркером.
func (s *PostgresStore) GetOrdersToProcess(ctx context.Context, limit int) ([]Order, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, user_id, number, status, accrual, uploaded_at 
		 FROM service_data.orders 
		 WHERE status = $1 OR status = $2 
		 ORDER BY uploaded_at ASC 
		 LIMIT $3`,
		model.OrderStatusNew, model.OrderStatusProcessing, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения заказов для обработки: %w", err)
	}
	defer rows.Close()

	var orders []Order
	for rows.Next() {
		var order Order
		if err := rows.Scan(&order.ID, &order.UserID, &order.Number, &order.Status, &order.Accrual, &order.UploadedAt); err != nil {
			return nil, fmt.Errorf("ошибка чтения строки: %w", err)
		}
		orders = append(orders, order)
	}
	return orders, rows.Err()
}

// AccrueBalance начисляет баллы атомарно.
func (s *PostgresStore) AccrueBalance(ctx context.Context, userID, orderNumber string, accrual float64) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("не удалось начать транзакцию: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx,
		`UPDATE service_data.orders SET status = $1, accrual = $2 WHERE number = $3 AND user_id = $4`,
		model.OrderStatusProcessed, accrual, orderNumber, userID,
	)
	if err != nil {
		return fmt.Errorf("ошибка обновления заказа: %w", err)
	}

	_, err = tx.Exec(ctx,
		`UPDATE service_data.users SET balance = balance + $1 WHERE id = $2`,
		accrual, userID,
	)
	if err != nil {
		return fmt.Errorf("ошибка начисления баллов: %w", err)
	}

	return tx.Commit(ctx)
}

// WithdrawBalance списывает баллы и создаёт запись о выводе.
func (s *PostgresStore) WithdrawBalance(ctx context.Context, userID, orderNumber string, sum float64) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("не удалось начать транзакцию: %w", err)
	}
	defer tx.Rollback(ctx)

	var newBalance float64
	err = tx.QueryRow(ctx,
		`UPDATE service_data.users 
		 SET balance = balance - $1, withdrawn = withdrawn + $1 
		 WHERE id = $2 AND balance >= $1 
		 RETURNING balance`,
		sum, userID,
	).Scan(&newBalance)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrInsufficientBalance
		}
		return fmt.Errorf("ошибка списания: %w", err)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO service_data.withdrawals (user_id, order_number, sum) VALUES ($1, $2, $3)`,
		userID, orderNumber, sum,
	)
	if err != nil {
		return fmt.Errorf("ошибка создания записи вывода: %w", err)
	}

	return tx.Commit(ctx)
}

// GetUserWithdrawals возвращает все выводы средств пользователя.
func (s *PostgresStore) GetUserWithdrawals(ctx context.Context, userID string) ([]Withdrawal, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, user_id, order_number, sum, processed_at 
		 FROM service_data.withdrawals WHERE user_id = $1 
		 ORDER BY processed_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения выводов: %w", err)
	}
	defer rows.Close()

	var withdrawals []Withdrawal
	for rows.Next() {
		var w Withdrawal
		if err := rows.Scan(&w.ID, &w.UserID, &w.OrderNumber, &w.Sum, &w.ProcessedAt); err != nil {
			return nil, fmt.Errorf("ошибка чтения строки: %w", err)
		}
		withdrawals = append(withdrawals, w)
	}
	return withdrawals, rows.Err()
}

// Ping проверяет доступность базы данных.
func (s *PostgresStore) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

// Close закрывает пул соединений.
func (s *PostgresStore) Close() {
	s.pool.Close()
}

// isUniqueViolation проверяет, является ли ошибкой нарушения уникальности PostgreSQL.
// Использует pgconn.PgError для надёжной проверки по коду ошибки "23505".
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}
