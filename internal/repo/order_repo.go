package repo

import (
	"context"
	"errors"
	"fmt"
	"log"
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
	log.Printf("[OrderRepository] Increment order count for product with id=%d", productID)

	key := fmt.Sprintf("product_id: %d", productID)
	newValue, err := r.client.Incr(ctx, key).Result()
	if err != nil {
		log.Printf("[OrderRepository] Error incrementing product count for product with id=%d", productID)
		return 0, fmt.Errorf("failed to increment order counter: %w", err)
	}

	err = r.client.Expire(ctx, key, 5*time.Second).Err()
	if err != nil {
		log.Printf("[OrderRepository] Error setting TTL for key %s: %v", key, err)
		return 0, fmt.Errorf("Warning: failed to set TTL for key %s: %v\n", key, err)
	}

	log.Printf("[OrderRepository] Successfully incremented count of product %d. New value: %d", productID, newValue)
	return newValue, nil
}

func (r *OrderRepository) GetOrder(ctx context.Context, productID int) (int64, error) {
	log.Printf("[OrderRepository] Get order count for product with id=%d", productID)
	key := fmt.Sprintf("product_id: %d", productID)

	value, err := r.client.Get(ctx, key).Result()

	if errors.Is(err, redis.Nil) { // если ключа нет в базе редис (товар не покупали или запись удалена по TTL)
		log.Printf("[OrderRepository] Product with id %d does not exist", productID)
		return 0, nil
	} else if err != nil {
		log.Printf("[OrderRepository] Error getting product with id %d: %v", productID, err)
		return 0, fmt.Errorf("failed to get order counter: %w", err)
	}

	count, err := strconv.ParseInt(value, 10, 64)

	if err != nil {
		log.Printf("[OrderRepository] Error converting order counter to integer: %v", err)
		return 0, fmt.Errorf("failed to convert order counter: %w", err)
	}

	log.Printf("[OrderRepository] Successfully got order count for product with id=%d", productID)
	return count, nil
}
