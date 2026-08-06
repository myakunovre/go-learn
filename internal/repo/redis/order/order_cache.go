package order

import (
	"context"
	"errors"
	"fmt"
	"go-learn/internal/service/order"
	"log/slog"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type OrderCacheRepository struct {
	client *redis.Client
	logger *slog.Logger
}

func NewOrderCacheRepository(client *redis.Client, logger *slog.Logger) *OrderCacheRepository {
	return &OrderCacheRepository{client: client, logger: logger}
}

func (r *OrderCacheRepository) IncrementOrder(ctx context.Context, productID int64) (int64, error) {
	r.logger.Debug("[OrderCache] Incrementing order", "productID", productID)

	key := fmt.Sprintf("product_id: %d", productID)
	newValue, err := r.client.Incr(ctx, key).Result()
	if err != nil {
		r.logger.Error("[OrderCache] Error incrementing order", "productID", productID, "err", err)
		return 0, fmt.Errorf("failed to increment order counter: %w", err)
	}

	err = r.client.Expire(ctx, key, 5*time.Second).Err()
	if err != nil {
		r.logger.Error("[OrderCache] Error expire order", "productID", productID, "err", err)
		return 0, fmt.Errorf("Warning: failed to set TTL for key %s: %w\n", key, err)
	}

	r.logger.Info("[OrderCache] Success incrementing order", "productID", productID, "newValue", newValue)
	return newValue, nil
}

func (r *OrderCacheRepository) GetOrder(ctx context.Context, productID int64) (int64, error) {
	r.logger.Debug("[OrderCache] Get order count", "productID", productID)

	key := fmt.Sprintf("product_id: %d", productID)
	value, err := r.client.Get(ctx, key).Result()

	if errors.Is(err, redis.Nil) { // если ключа нет в базе редис (товар не покупали или запись удалена по TTL)
		r.logger.Warn("[OrderCache] Product does not exist", "productID", productID)
		return 0, nil
	} else if err != nil {
		r.logger.Error("[OrderCache] Error getting order", "productID", productID, "err", err)
		return 0, fmt.Errorf("failed to get order counter: %w", err)
	}

	count, err := strconv.ParseInt(value, 10, 64)

	if err != nil {
		r.logger.Error("[OrderCache] Error parsing order value", "productID", productID, "value", value)
		return 0, fmt.Errorf("failed to convert order counter: %w", err)
	}

	r.logger.Info("[OrderCache] Success get order count", "productID", productID)
	return count, nil
}

func (r *OrderCacheRepository) DeleteProductFromOrder(ctx context.Context, productID int64) error {
	r.logger.Debug("[OrderCache] Deleting core", "productID", productID)

	key := fmt.Sprintf("product_id: %d", productID)
	_, err := r.client.Get(ctx, key).Result()

	if errors.Is(err, redis.Nil) { // если ключа нет в базе редис (товар не покупали или запись удалена по TTL)
		r.logger.Warn("[OrderCache] Product does not exist", "productID", productID)
		return nil
	} else if err != nil {
		r.logger.Error("[OrderCache] Error getting order", "productID", productID, "err", err)
		return fmt.Errorf("failed to get order core: %w", err)
	}

	_, err = r.client.Del(ctx, key).Result()
	if err != nil {
		r.logger.Error("[OrderCache] Error deleting core from order", "productID", productID, "err", err)
		return fmt.Errorf("failed to deleting order counter: %w", err)
	}

	r.logger.Info("[OrderCache] Success deleting core from order cache", "productID", productID)
	return nil
}

// func (r *OrderCacheRepository) GetAllOrders(ctx context.Context) (map[int64]int64, error) {
func (r *OrderCacheRepository) GetAllOrders(ctx context.Context) ([]order.OrderProduct, error) {
	r.logger.Debug("[OrderCache] GetAll orders count")

	keys, err := r.client.Keys(ctx, "product_id: *").Result()
	if err != nil {
		r.logger.Error("[OrderCache] Error getting keys", "err", err)
		return nil, fmt.Errorf("failed to get order keys: %w", err)
	}

	if len(keys) == 0 {
		r.logger.Warn("[OrderCache] No products found in cache")
		return []order.OrderProduct{}, nil
	}

	// Получаем все значения одним запросом
	values, err := r.client.MGet(ctx, keys...).Result()
	if err != nil {
		r.logger.Error("[OrderCache] Error getting values from redis", "err", err)
		return nil, fmt.Errorf("failed to get order values: %w", err)
	}

	var orderProducts []order.OrderProduct
	for i, key := range keys {
		// Парсим productID из ключа
		var productId int64
		_, err = fmt.Sscanf(key, "%d", &productId)
		if err != nil {
			r.logger.Error("[OrderCache] Error parsing key", "key", key, "err", err)
			continue
		}

		if values[i] == nil {
			r.logger.Error("[OrderCache] Error parsing values", "key", key, "value", values[i])
			continue
		}

		// Преобразуем значение в строку
		valueStr, ok := values[i].(string)
		if !ok {
			r.logger.Error("[OrderCache] Error parsing value", "key", key, "value", values[i])
			continue
		}

		// Преобразуем значение из строки в число
		quantity, err := strconv.Atoi(valueStr)
		if err != nil {
			r.logger.Error("[OrderCache] Error parsing quantity", "key", key, "value", values[i])
			continue
		}

		orderProducts = append(orderProducts, order.OrderProduct{
			ProductId: int(productId),
			Quantity:  quantity,
		})
	}
	// todo: удалить закомментированный код
	//values, err := r.client.MGet(ctx, keys...).Result()
	//if err != nil {
	//	r.logger.Error("[OrderCache] Error getting values", "err", err)
	//	return nil, fmt.Errorf("failed to get order values: %w", err)
	//}
	//
	//orders := make(map[int64]int64, len(keys))
	//
	//for i, key := range keys {
	//	// Извлекаем productID из ключа "product_id: 123"
	//	var productID int64
	//	_, err = fmt.Sscanf(key, "product_id: %d", &productID)
	//	if err != nil {
	//		r.logger.Error("[OrderCache] Error parsing key", "key", key, "err", err)
	//		continue
	//	}
	//
	//	// Проверяем, что значение (count) не nil
	//	if values[i] == nil {
	//		r.logger.Warn("[OrderCache] Nil value for key", "key", key)
	//		continue
	//	}
	//
	//	// Преобразуем значение в int64
	//	valueStr, ok := values[i].(string)
	//	if !ok {
	//		r.logger.Error("[OrderCache] Invalid value type for key", "key", key, "value", values[i])
	//		continue
	//	}
	//
	//	count, err := strconv.ParseInt(valueStr, 10, 64)
	//	if err != nil {
	//		r.logger.Warn("[OrderCache] Error parsing core count", "key", key, "value", values[i])
	//		continue
	//	}
	//
	//	orders[productID] = count
	//}

	r.logger.Info("[OrderCache] Success get all orders", "count", len(orderProducts))
	return orderProducts, nil
}
