package order

import (
	"context"
	"errors"
	"fmt"
	"go-learn/api/grpc/proto/pb"
	"go-learn/internal/domain/order"
	"go-learn/models"
	"log/slog"
	"time"
)

type OrderRepository interface {
	FindOrderIDByUser(ctx context.Context, userId int64) (int, bool, error)
	CreateOrder(ctx context.Context, description string, userId int64, products []order.Product, deliveryTimeHours int) (int, error)
	MergeOrder(ctx context.Context, orderId int, products []order.Product, deliveryTimeHours int) error
	GetOrder(ctx context.Context, id int) (*models.Order, error)
	DeleteOrder(ctx context.Context, id int) error
	MarkDeletedProduct(ctx context.Context, productId int) error
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

type CreateOrderInput struct {
	Description string
	UserID      int64
	Products    []OrderProduct
}

type OrderProduct struct {
	ProductID int64
	Quantity  int
}

const deliveryHoursPerPosition = 12

func calculateDeliveryTimeHours(positionCount int) int {
	return deliveryHoursPerPosition * positionCount
}

func (s *OrderService) Create(ctx context.Context, input CreateOrderInput) (int, error) {
	if input.Description == "" {
		s.logger.Error("[OrderService] Order description is blank", "description", input.Description)
		return 0, errors.New("order description is blank")
	}

	if input.UserID <= 0 {
		s.logger.Error("[OrderService] userId less than zero", "userId", input.UserID)
		return 0, errors.New("userId less than zero")
	}

	if len(input.Products) == 0 {
		s.logger.Error("[OrderService] No orderProducts in order")
		return 0, errors.New("no orderProducts in order")
	}

	// Создаем мапу товар: кол-во (объединяем одинаковые позиции одного входящего заказа)
	quantityMap := make(map[int64]int64)

	for _, product := range input.Products {
		quantityMap[product.ProductID] += int64(product.Quantity)
	}

	// Рассчитываем время доставки (0.5 дня за каждую уникальную товарную позицию)
	deliveryTimeHours := calculateDeliveryTimeHours(len(quantityMap))

	// Создаем слайс уникальных ID товаров для запроса из core-сервиса по gRPC
	productIDs := make([]int64, 0, len(quantityMap))

	for productID := range quantityMap {
		productIDs = append(productIDs, productID)
	}

	// Получаем товары из core-сервиса по gRPC
	grpcCtx, cancel := context.WithTimeout(ctx, 7*time.Second)
	defer cancel()

	productsFromCore, err := s.coreGRPCClient.GetProducts(grpcCtx, productIDs)

	if err != nil {
		s.logger.Error("[OrderService] fail to get products from core", "error", err)
		return 0, errors.New("fail to get products from core")
	}

	// Получаем слайс Products из запроса
	products := productsFromCore.GetProducts()

	// Создаем слайс order.Product для отправки в OrderRepo
	productsToOrder := make([]order.Product, 0, len(products))

	for _, product := range products {
		quantity, exists := quantityMap[product.Id]
		if !exists {
			continue
		}

		productsToOrder = append(
			productsToOrder,
			order.Product{
				ProductId:            product.Id,
				ProductName:          product.Name,
				ProductAmountInCore:  int32(product.Amount),
				ProductAmountInOrder: int32(quantity),
				ProductPrice:         int32(product.Price),
				ItemExists:           true,
			},
		)
	}

	dbCtx, cancel := context.WithTimeout(ctx, 7*time.Second)
	defer cancel()

	// Проверяем есть ли у пользователя уже заказы
	existingOrderID, isFound, err := s.repo.FindOrderIDByUser(dbCtx, input.UserID)

	if err != nil {
		s.logger.Error("[OrderService] fail to find existing user order", "userId", input.UserID, "error", err)
		return 0, fmt.Errorf("fail to find existing user order: %w", err)
	}

	// Если у пользователя уже заказы есть, то объединяем, если нет, создаем новый
	if !isFound {
		id, err := s.repo.CreateOrder(
			dbCtx,
			input.Description,
			input.UserID,
			productsToOrder,
			deliveryTimeHours,
		)

		if err != nil {
			s.logger.Error("[OrderService] Error of creating order", "description", input.Description, "error", err)
			return 0, fmt.Errorf("order creation failed: %w", err)
		}

		s.logger.Info("[OrderService] ✅ Order created successfully", "id", id, "description", input.Description, "userId", input.UserID)
		return id, nil
	}

	err = s.repo.MergeOrder(
		dbCtx,
		existingOrderID,
		productsToOrder,
		deliveryTimeHours,
	)

	if err != nil {
		s.logger.Error("[OrderService] fail to merge order",
			"orderId", existingOrderID,
			"userId", input.UserID,
			"error", err,
		)

		return 0, fmt.Errorf("fail to merge order: %w", err)
	}

	s.logger.Info(
		"[OrderService] Order merged successfully",
		"orderId", existingOrderID,
		"userId", input.UserID,
	)

	return existingOrderID, nil
}

func (s *OrderService) Get(ctx context.Context, id int) (*models.Order, error) {
	if id <= 0 {
		s.logger.Warn("order id should be greater than zero")
		return nil, errors.New("order id should be greater than zero")
	}

	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	order, err := s.repo.GetOrder(dbCtx, id)
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

	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err := s.repo.DeleteOrder(dbCtx, id)
	if err != nil {
		s.logger.Error("[OrderService] Error of deleting order", "id", id, "error", err)
		return fmt.Errorf("failed to delete order with id %d: %w", id, err)
	}

	s.logger.Info("[OrderService] ✅ Order deleted successfully", "id", id)
	return nil
}

// MarkDeletedProduct маркирует удаленные товары во всех заказах (order_items)
func (s *OrderService) MarkDeletedProduct(ctx context.Context, productId int) error {
	if productId <= 0 {
		s.logger.Warn("core id should be greater than zero")
		return errors.New("core id should be greater than zero")
	}

	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err := s.repo.MarkDeletedProduct(dbCtx, productId)
	if err != nil {
		s.logger.Error("[OrderService] Error marking core", "productId", productId, "error", err)
		return fmt.Errorf("failed to mark deleted core with id %d: %w", productId, err)
	}

	return nil
}
