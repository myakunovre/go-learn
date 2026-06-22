package order

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type OrderRepository struct {
	client *redis.Client
	logger *slog.Logger
}

func NewOrderRepository(client *redis.Client, logger *slog.Logger) *OrderRepository {
	return &OrderRepository{client: client, logger: logger}
}

func (r *OrderRepository) IncrementOrder(ctx context.Context, productID int) (int64, error) {
	r.logger.Debug("[OrderRepository] Incrementing order", "productID", productID)

	key := fmt.Sprintf("product_id: %d", productID)
	newValue, err := r.client.Incr(ctx, key).Result()
	if err != nil {
		r.logger.Error("[OrderRepository] Error incrementing order", "productID", productID, "err", err)
		return 0, fmt.Errorf("failed to increment order counter: %w", err)
	}

	err = r.client.Expire(ctx, key, 5*time.Second).Err()
	if err != nil {
		r.logger.Error("[OrderRepository] Error expire order", "productID", productID, "err", err)
		return 0, fmt.Errorf("Warning: failed to set TTL for key %s: %w\n", key, err)
	}

	r.logger.Info("[OrderRepository] Success incrementing order", "productID", productID, "newValue", newValue)
	return newValue, nil
}

func (r *OrderRepository) GetOrder(ctx context.Context, productID int) (int64, error) {
	r.logger.Debug("[OrderRepository] Get order count", "productID", productID)

	key := fmt.Sprintf("product_id: %d", productID)
	value, err := r.client.Get(ctx, key).Result()

	if errors.Is(err, redis.Nil) { // если ключа нет в базе редис (товар не покупали или запись удалена по TTL)
		r.logger.Warn("[OrderRepository] Product does not exist", "productID", productID)
		return 0, nil
	} else if err != nil {
		r.logger.Error("[OrderRepository] Error getting order", "productID", productID, "err", err)
		return 0, fmt.Errorf("failed to get order counter: %w", err)
	}

	count, err := strconv.ParseInt(value, 10, 64)

	if err != nil {
		r.logger.Error("[OrderRepository] Error parsing order value", "productID", productID, "value", value)
		return 0, fmt.Errorf("failed to convert order counter: %w", err)
	}

	r.logger.Info("[OrderRepository] Success get order count", "productID", productID)
	return count, nil
}
