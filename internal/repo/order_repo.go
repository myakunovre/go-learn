package repo

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type OrderRepository struct {
	client *redis.Client
}

func NewOrderRepository(client *redis.Client) *OrderRepository {
	return &OrderRepository{client: client}
}

func (r *OrderRepository) IncrementOrder(ctx context.Context, productID int) (int64, error) {
	key := fmt.Sprintf("product_id: %d", productID)
	newValue, err := r.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to increment order counter: %w", err)
	}

	err = r.client.Expire(ctx, key, 5*time.Second).Err()
	if err != nil {
		return 0, fmt.Errorf("Warning: failed to set TTL for key %s: %v\n", key, err)
	}

	return newValue, nil
}

func (r *OrderRepository) GetOrder(ctx context.Context, productID int) (int64, error) {
	key := fmt.Sprintf("product_id: %d", productID)

	value, err := r.client.Get(ctx, key).Result()

	if errors.Is(err, redis.Nil) { // если ключа нет в базе редис (товар не покупали или запись удалена по TTL)
		return 0, nil
	} else if err != nil {
		return 0, fmt.Errorf("failed to get order counter: %w", err)
	}

	count, err := strconv.ParseInt(value, 10, 64)

	if err != nil {
		return 0, fmt.Errorf("failed to convert order counter: %w", err)
	}

	return count, nil
}
