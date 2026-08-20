-- +goose Up

ALTER TABLE orders
ADD COLUMN delivery_time_hours INTEGER NOT NULL DEFAULT 0
CHECK (delivery_time_hours >= 0);

CREATE INDEX IF NOT EXISTS idx_orders_user_id
ON orders(user_id);


-- +goose Down

DROP INDEX IF EXISTS idx_orders_user_id;

ALTER TABLE orders
DROP COLUMN delivery_time_hours;