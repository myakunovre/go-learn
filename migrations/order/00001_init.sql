-- +goose Up
CREATE TABLE IF NOT EXISTS orders (
    id SERIAL PRIMARY KEY,
    external_id INTEGER UNIQUE,
    cost INTEGER NOT NULL CHECK (cost >= 0),
    amount INTEGER NOT NULL CHECK (amount >= 0),
    name VARCHAR(255) NOT NULL,
    user_id INTEGER
    );

CREATE TABLE IF NOT EXISTS sessions (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token VARCHAR(255) NOT NULL UNIQUE,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
    );

-- +goose Down
DROP TABLE IF EXISTS orders;
DROP TABLE IF EXISTS sessions;
