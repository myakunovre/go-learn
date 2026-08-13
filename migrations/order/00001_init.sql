-- +goose Up
CREATE TABLE IF NOT EXISTS orders (
    id SERIAL PRIMARY KEY,
    description VARCHAR(255),
    user_id INTEGER
);

CREATE TABLE IF NOT EXISTS order_items (
    id SERIAL PRIMARY KEY,
    order_id INTEGER NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id INTEGER NOT NULL,
    product_name VARCHAR(255) NOT NULL,
    product_amount_in_core INTEGER NOT NULL CHECK (product_amount_in_core > 0),
    product_amount_in_order INTEGER NOT NULL CHECK (product_amount_in_order > 0),
    product_price INTEGER NOT NULL,
    item_exists BOOLEAN DEFAULT TRUE,
    UNIQUE (order_id, product_id)
);

-- CREATE TABLE IF NOT EXISTS sessions (
--     id SERIAL PRIMARY KEY,
--     user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
--     token VARCHAR(255) NOT NULL UNIQUE,
--     expires_at TIMESTAMP NOT NULL,
--     created_at TIMESTAMP DEFAULT NOW()
-- );

-- +goose Down
DROP TABLE IF EXISTS orders;
-- DROP TABLE IF EXISTS sessions;
