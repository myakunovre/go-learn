package order

import (
	"context"
	"errors"
	"fmt"
	"go-learn/api/grpc/proto/pb"
	"go-learn/internal/domain/order"
	"go-learn/models"
	"log/slog"
)

type OrderRepository interface {
	CreateOrder(description string, userId int, products []order.Product) (int, error)
	GetOrder(id int) (*models.Order, error)
	DeleteOrder(id int) error
	MarkDeletedProduct(productId int) error
}

type CoreGRPCClient interface {
	GetProducts(ctx context.Context, productIDs []int64) (*pb.ProductResponse, error)
}

type OrderService struct {
	repo           OrderRepository
	coreGRPCClient CoreGRPCClient
	logger         *slog.Logger
}

func NewOrderService(repo OrderRepository, coreGRPCClient CoreGRPCClient, logger *slog.Logger) *OrderService {
	return &OrderService{
		repo:           repo,
		coreGRPCClient: coreGRPCClient,
		logger:         logger}
}

// todo: перенести структуру в модели хэндлера
type OrderProduct struct {
	ProductId int
	Quantity  int
}

func (s *OrderService) Create(ctx context.Context, description string, userId int, orderProducts []OrderProduct) (int, error) {
	if len(description) == 0 {
		s.logger.Error("[OrderService] Order description is blank", "description", description)
		return 0, errors.New("order description is blank")
	}

	if userId <= 0 {
		s.logger.Error("[OrderService] userId less than zero", "userId", userId)
		return 0, errors.New("userId less than zero")
	}

	if len(orderProducts) == 0 {
		s.logger.Error("[OrderService] No orderProducts in order")
		return 0, errors.New("no orderProducts in order")
	}

	// todo: проверить этот блок кода (получение товаров по gRPC из core-сервиса)
	// Создаем слайс с ID товаров для запроса из core-сервиса по gRPC
	productIDs := make([]int64, 0, len(orderProducts))
	//quantities := make([]int64, 0, len(orderProducts))
	for _, product := range orderProducts {
		productIDs = append(productIDs, int64(product.ProductId))
		//quantities = append(quantities, int64(product.Quantity))
	}

	// Получаем товары по ID из core-сервиса по gRPC
	productsFromCore, err := s.coreGRPCClient.GetProducts(context.Background(), productIDs)
	if err != nil {
		s.logger.Error("[OrderService] fail to get products from core", "error", err)
		return 0, errors.New("fail to get products from core")
	}

	quantityMap := make(map[int64]int64)
	for _, product := range orderProducts {
		quantityMap[int64(product.ProductId)] = int64(product.Quantity)
	}

	// Получаем слайс Products из запроса
	products := productsFromCore.GetProducts()

	// Создаем слайс order.Product для отправки в OrderRepo
	var productsToOrder []order.Product
	for _, product := range products {
		quantity, exists := quantityMap[product.Id]
		if !exists {
			continue
		}
		productsToOrder = append(productsToOrder, order.Product{
			ProductId:            product.Id,
			ProductName:          product.Name,
			ProductAmountInCore:  int32(product.Amount),
			ProductAmountInOrder: int32(quantity),
			ProductPrice:         int32(product.Price),
			ItemExists:           true,
		})
	}

	id, err := s.repo.CreateOrder(description, userId, productsToOrder)
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

func (s *OrderService) Delete(ctx context.Context, id int) error {
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
		s.logger.Warn("core id should be greater than zero")
		return errors.New("core id should be greater than zero")
	}

	err := s.repo.MarkDeletedProduct(productId)
	if err != nil {
		s.logger.Error("[OrderService] Error marking core", "productId", productId, "error", err)
		return fmt.Errorf("failed to mark deleted core with id %d: %w", productId, err)
	}

	return nil
}
