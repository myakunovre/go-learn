package order

import (
	"errors"
	"fmt"
	"go-learn/models"
	"log/slog"
)

type OrderRepository interface {
	CreateOrder(description string, userId int, products map[int]int) (int, error)
	GetOrder(id int) (*models.Order, error)
	DeleteOrder(id int) error
	MarkDeletedProduct(productId int) error
}

type OrderService struct {
	repo   OrderRepository
	logger *slog.Logger
}

func NewOrderService(repo OrderRepository, logger *slog.Logger) *OrderService {
	return &OrderService{
		repo:   repo,
		logger: logger,
	}
}

func (s *OrderService) Create(description string, userId int, products map[int]int) (int, error) {
	if len(description) == 0 {
		s.logger.Error("[OrderService] Order description is blank", "description", description)
		return 0, errors.New("order description is blank")
	}

	if userId <= 0 {
		s.logger.Error("[OrderService] userId less than zero", "userId", userId)
		return 0, errors.New("userId less than zero")
	}

	if len(products) == 0 {
		s.logger.Error("[OrderService] No products in order")
		return 0, errors.New("no products in order")
	}

	id, err := s.repo.CreateOrder(description, userId, products)
	if err != nil {
		s.logger.Error("[OrderService] Error of creating order", "description", description, "error", err)
		return 0, fmt.Errorf("order creation failed: %w", err)
	}

	s.logger.Info("[OrderService] ✅ Order created successfully", "id", id, "description", description, "userId", userId)
	return id, nil
}

func (s *OrderService) Get(id int) (*models.Order, error) {
	if id <= 0 {
		s.logger.Warn("order id should be greater than zero")
		return nil, errors.New("order id should be greater than zero")
	}

	order, err := s.repo.GetOrder(id)
	if err != nil {
		s.logger.Error("[OrderService] Error of getting order", "id", id, "error", err)
		return nil, fmt.Errorf("order get failed: %w", err)
	}

	s.logger.Info("[OrderService] ✅ Order got successful", "id", id, "description", order.Description)
	return order, nil
}

func (s *OrderService) Delete(id int) error {
	if id <= 0 {
		s.logger.Warn("order id should be greater than zero")
		return errors.New("order id should be greater than zero")
	}

	err := s.repo.DeleteOrder(id)
	if err != nil {
		s.logger.Error("[OrderService] Error of deleting order", "id", id, "error", err)
		return fmt.Errorf("failed to delete order with id %d: %w", id, err)
	}

	s.logger.Info("[OrderService] ✅ Order deleted successfully", "id", id)
	return nil
}

// MarkDeletedProduct маркирует удаленные товары во всех заказах (order_items)
func (s *OrderService) MarkDeletedProduct(productId int) error {
	if productId <= 0 {
		s.logger.Warn("product id should be greater than zero")
		return errors.New("product id should be greater than zero")
	}

	err := s.repo.MarkDeletedProduct(productId)
	if err != nil {
		s.logger.Error("[OrderService] Error marking product", "productId", productId, "error", err)
		return fmt.Errorf("failed to mark deleted product with id %d: %w", productId, err)
	}

	return nil
}
