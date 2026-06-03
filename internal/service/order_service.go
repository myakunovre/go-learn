package service

import (
	"context"
	"fmt"
)

type OrderRepository interface {
	IncrementOrder(ctx context.Context, productID int) (int64, error)
	GetOrder(ctx context.Context, productID int) (int64, error)
}

type OrderService struct {
	repo OrderRepository
}

func NewOrderService(repo OrderRepository) *OrderService {
	return &OrderService{repo: repo}
}

func (service *OrderService) BuyProduct(ctx context.Context, productID int) (int64, error) {
	totalOrders, err := service.repo.IncrementOrder(ctx, productID)
	if err != nil {
		return 0, fmt.Errorf("failed to increment order: %w", err)
	}

	return totalOrders, nil
}

func (service *OrderService) GetOrderCount(ctx context.Context, productID int) (int64, error) {
	count, err := service.repo.GetOrder(ctx, productID)
	if err != nil {
		return 0, fmt.Errorf("failed to get order: %w", err)
	}

	return count, nil
}
