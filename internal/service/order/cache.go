package order

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
)

type OrderCacheRepository interface {
	IncrementOrder(ctx context.Context, productID int64) (int64, error)
	GetOrder(ctx context.Context, productID int64) (int64, error)
	DeleteProductFromOrder(ctx context.Context, productID int64) error
	GetAllOrders(ctx context.Context) (map[int64]int64, error)
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
		s.logger.Warn("product id should be greater than zero")
		return 0, errors.New("product id should be greater than zero")
	}

	totalOrders, err := s.repo.IncrementOrder(ctx, productID)
	if err != nil {
		s.logger.Error("[OrderCacheService] Error increment order for product", "id", productID, "err", err)
		return 0, fmt.Errorf("failed to increment order: %w", err)
	}

	s.logger.Info("[OrderCacheService] Success increment order for product", "id", productID)
	return totalOrders, nil
}

func (s *OrderCacheService) GetOrderCount(ctx context.Context, productID int64) (int64, error) {
	if productID <= 0 {
		s.logger.Warn("[OrderCacheService] Product id should be greater than zero")
		return 0, errors.New("product id should be greater than zero")
	}

	count, err := s.repo.GetOrder(ctx, productID)
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
		return errors.New("product id should be greater than zero")
	}

	err := s.repo.DeleteProductFromOrder(ctx, productID)
	if err != nil {
		s.logger.Error("[OrderCacheService] Error delete product", "id", productID, "err", err)
		return fmt.Errorf("failed to delete product: %w", err)
	}

	s.logger.Info("[OrderCacheService] Success delete product", "id", productID)
	return nil
}

func (s *OrderCacheService) GetAllProducts(ctx context.Context) (map[int64]int64, error) {
	products, err := s.repo.GetAllOrders(ctx)
	if err != nil {
		s.logger.Error("[OrderCacheService] Error get all products", "err", err)
		return nil, fmt.Errorf("failed to get all products: %w", err)
	}

	return products, nil
}
