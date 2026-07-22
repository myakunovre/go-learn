package product

import (
	"context"
	"errors"
	"fmt"
	"go-learn/internal/events"
	"go-learn/models"
	"log/slog"
)

type ProductRepository interface {
	DeleteProduct(int) error
	CreateProduct(name string, price int) (int, error)
	GetProduct(int) (*models.Product, error)
	GetAllProducts() ([]models.Product, error)
}

type ProductService struct {
	repo      ProductRepository
	logger    *slog.Logger
	publisher events.EventPublisher
}

func NewProductService(repo ProductRepository, logger *slog.Logger, publisher events.EventPublisher) *ProductService {
	return &ProductService{
		repo:      repo,
		logger:    logger,
		publisher: publisher,
	}
}

func (s *ProductService) Create(ctx context.Context, name string, price int) (int, error) {
	if price <= 0 {
		s.logger.Error("[ProductService] Price less than zero", "price", price)
		return 0, errors.New("price less than zero")
	}

	if len(name) == 0 {
		s.logger.Error("[ProductService] Price less than zero", "price", price)
		return 0, errors.New("length of name is zero")
	}

	id, err := s.repo.CreateProduct(name, price)
	if err != nil {
		s.logger.Error("[ProductService] Error of creating product", "name", name, "error", err)
		return 0, fmt.Errorf("product creation failed: %w", err)
	}

	// Публикуем событие о создании товара через KAFKA
	event := events.ProductCreated{
		ProductID: id,
		Name:      name,
		Price:     price,
	}
	if pubErr := s.publisher.Publish(ctx, "product-events", event); pubErr != nil {
		s.logger.Warn("Failed to publish ProductCreated event", "error", pubErr)
		// Не возвращаем ошибку, чтобы не нарушать основную операцию
	}

	s.logger.Info("[ProductService] ✅ Product created successfully", "id", id, "name", name, "price", price)
	return id, nil
}

func (s *ProductService) Get(id int) (*models.Product, error) {
	if id <= 0 {
		s.logger.Warn("product id should be greater than zero")
		return nil, errors.New("product id should be greater than zero")
	}

	product, err := s.repo.GetProduct(id)
	if err != nil {
		s.logger.Error("[ProductService] Error of getting product", "id", id, "error", err)
		return nil, fmt.Errorf("product get failed: %w", err)
	}

	s.logger.Info("[ProductService] ✅ Product got successful", "id", id, "name", product.Name)
	return product, nil
}

func (s *ProductService) GetAllProducts() ([]models.Product, error) {
	products, err := s.repo.GetAllProducts()
	if err != nil {
		s.logger.Error("[ProductService] Error of getting all products", "error", err)
		return nil, fmt.Errorf("product get failed: %w", err)
	}

	s.logger.Info("[ProductService] All Products got successful", "num", len(products))
	return products, nil
}

func (s *ProductService) Delete(ctx context.Context, id int) error {
	if id <= 0 {
		s.logger.Warn("product id should be greater than zero")
		return errors.New("product id should be greater than zero")
	}

	err := s.repo.DeleteProduct(id)
	if err != nil {
		s.logger.Error("[ProductService] Error of deleting product", "id", id, "error", err)
		return fmt.Errorf("failed to delete product with id %d: %v", id, err)
	}

	//анализируем цену товара
	product, err := s.repo.GetProduct(id)
	if err != nil {
		s.logger.Error("[ProductService] Error of getting product", "id", id, "error", err)
	}
	price := product.Price

	if price < 1000 {
		s.logger.Info("[ProductService] ✅ Product deleted successfully", "id", id)
		return nil
	}

	if price > 10000 {
		//todo отменить удаление (транзакция)
		s.logger.Info("[ProductService] ✅ Product not deleted, price must be smaller than 10000", "id", id)
		return nil
	}

	// Публикуем событие об удалении товара через KAFKA
	event := events.ProductDeleted{ProductID: id}
	if pubErr := s.publisher.Publish(ctx, "product-events", event); pubErr != nil {
		s.logger.Warn("Failed to publish ProductDeleted event", "error", pubErr)
	}

	s.logger.Info("[ProductService] ✅ Product deleted successfully", "id", id)
	return nil
}
