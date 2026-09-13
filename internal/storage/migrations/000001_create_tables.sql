-- +goose Up
CREATE TABLE IF NOT EXISTS service_data.users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    login VARCHAR(255) NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    balance NUMERIC(10, 2) NOT NULL DEFAULT 0,
    withdrawn NUMERIC(10, 2) NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
    );

CREATE TABLE IF NOT EXISTS service_data.orders (
                                                   id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES service_data.users(id) ON DELETE CASCADE,
    number VARCHAR(255) NOT NULL UNIQUE,
    status VARCHAR(20) NOT NULL DEFAULT 'NEW',
    accrual NUMERIC(10, 2),
    uploaded_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
    );

CREATE INDEX IF NOT EXISTS idx_orders_user_id ON service_data.orders(user_id);
CREATE INDEX IF NOT EXISTS idx_orders_status ON service_data.orders(status);

CREATE TABLE IF NOT EXISTS service_data.withdrawals (
                                                        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES service_data.users(id) ON DELETE CASCADE,
    order_number VARCHAR(255) NOT NULL,
    sum NUMERIC(10, 2) NOT NULL,
    processed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
    );

CREATE INDEX IF NOT EXISTS idx_withdrawals_user_id ON service_data.withdrawals(user_id);

-- +goose Down
DROP TABLE IF EXISTS service_data.users;
DROP TABLE IF EXISTS service_data.orders;
DROP TABLE IF EXISTS service_data.withdrawals;
