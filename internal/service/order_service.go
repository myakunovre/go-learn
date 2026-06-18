package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
)

type OrderRepository interface {
	IncrementOrder(ctx context.Context, productID int) (int64, error)
	GetOrder(ctx context.Context, productID int) (int64, error)
}

type OrderService struct {
	repo   OrderRepository
	logger *slog.Logger
}

func NewOrderService(repo OrderRepository, logger *slog.Logger) *OrderService {
	return &OrderService{repo: repo, logger: logger}
}

func (s *OrderService) BuyProduct(ctx context.Context, productID int) (int64, error) {
	if productID <= 0 {
		s.logger.Warn("product id should be greater than zero")
		return 0, errors.New("product id should be greater than zero")
	}

	totalOrders, err := s.repo.IncrementOrder(ctx, productID)
	if err != nil {
		s.logger.Error("[OrderService] Error increment order for product", "id", productID, "err", err)
		return 0, fmt.Errorf("failed to increment order: %w", err)
	}

	s.logger.Info("[OrderService] Success increment order for product", "id", productID)
	return totalOrders, nil
}

func (s *OrderService) GetOrderCount(ctx context.Context, productID int) (int64, error) {
	if productID <= 0 {
		s.logger.Warn("product id should be greater than zero")
		return 0, errors.New("product id should be greater than zero")
	}

	count, err := s.repo.GetOrder(ctx, productID)
	if err != nil {
		s.logger.Error("[OrderService] Error get order count", "id", productID, "err", err)
		return 0, fmt.Errorf("failed to get order: %w", err)
	}

	s.logger.Info("[OrderService] Success get order count", "id", productID)
	return count, nil
}
