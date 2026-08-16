package order

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"
)

type OrderCacheRepository interface {
	IncrementOrder(ctx context.Context, productID int64) (int64, error)
	GetOrder(ctx context.Context, productID int64) (int64, error)
	DeleteProductFromOrder(ctx context.Context, productID int64) error
	GetAllOrders(ctx context.Context) ([]OrderProduct, error)
}

type OrderCacheService struct {
	repo   OrderCacheRepository
	logger *slog.Logger
}

func NewOrderCacheService(repo OrderCacheRepository, logger *slog.Logger) *OrderCacheService {
	return &OrderCacheService{repo: repo, logger: logger}
}

func (s *OrderCacheService) BuyProduct(ctx context.Context, productID int64) (int64, error) {
	if productID <= 0 {
		s.logger.Warn("core id should be greater than zero")
		return 0, errors.New("core id should be greater than zero")
	}

	cacheCtx, cacheCancel := context.WithTimeout(ctx, 2*time.Second)
	defer cacheCancel()

	totalOrders, err := s.repo.IncrementOrder(cacheCtx, productID)
	if err != nil {
		s.logger.Error("[OrderCacheService] Error increment order for core", "id", productID, "err", err)
		return 0, fmt.Errorf("failed to increment order: %w", err)
	}

	s.logger.Info("[OrderCacheService] Success increment order for core", "id", productID)
	return totalOrders, nil
}

func (s *OrderCacheService) GetOrderCount(ctx context.Context, productID int64) (int64, error) {
	if productID <= 0 {
		s.logger.Warn("[OrderCacheService] Product id should be greater than zero")
		return 0, errors.New("core id should be greater than zero")
	}

	cacheCtx, cacheCancel := context.WithTimeout(ctx, 2*time.Second)
	defer cacheCancel()

	count, err := s.repo.GetOrder(cacheCtx, productID)
	if err != nil {
		s.logger.Error("[OrderCacheService] Error get order count", "id", productID, "err", err)
		return 0, fmt.Errorf("failed to get order: %w", err)
	}

	s.logger.Info("[OrderService] Success get order count", "id", productID)
	return count, nil
}

func (s *OrderCacheService) DeleteProduct(ctx context.Context, productID int64) error {
	if productID <= 0 {
		s.logger.Warn("[OrderCacheService] Product id should be greater than zero")
		return errors.New("core id should be greater than zero")
	}

	cacheCtx, cacheCancel := context.WithTimeout(ctx, 2*time.Second)
	defer cacheCancel()

	err := s.repo.DeleteProductFromOrder(cacheCtx, productID)
	if err != nil {
		s.logger.Error("[OrderCacheService] Error delete core", "id", productID, "err", err)
		return fmt.Errorf("failed to delete core: %w", err)
	}

	s.logger.Info("[OrderCacheService] Success delete core", "id", productID)
	return nil
}

func (s *OrderCacheService) GetAllProducts(ctx context.Context) ([]OrderProduct, error) {
	cacheCtx, cacheCancel := context.WithTimeout(ctx, 2*time.Second)
	defer cacheCancel()

	products, err := s.repo.GetAllOrders(cacheCtx)
	if err != nil {
		s.logger.Error("[OrderCacheService] Error get all products", "err", err)
		return nil, fmt.Errorf("failed to get all products: %w", err)
	}

	return products, nil
}
