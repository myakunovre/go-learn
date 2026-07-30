package product

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"go-learn/internal/events"
	"go-learn/models"
	"log/slog"
)

type ProductRepository interface {
	DeleteProduct(int64) error
	DeleteProductWithTransaction(tx *sql.Tx, id int64) error
	CreateProduct(name string, price, amount int64) (int64, error)
	AddProduct(id, amount int64) (int64, error)
	GetProduct(int64) (*models.Product, error)
	GetAllProducts() ([]models.Product, error)
	NewTransaction(ctx context.Context) (*sql.Tx, error)
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

func (s *ProductService) Create(ctx context.Context, name string, price, amount int64) (int64, error) {
	if price <= 0 {
		s.logger.Error("[ProductService] Price less than zero", "price", price)
		return 0, errors.New("price less than zero")
	}

	if len(name) == 0 {
		s.logger.Error("[ProductService] Product name id blank", "name", name)
		return 0, errors.New("product name is blank")
	}

	if amount <= 0 {
		s.logger.Error("[ProductService] Amount less than zero", "amount", amount)
		return 0, errors.New("amount less than zero")
	}

	id, err := s.repo.CreateProduct(name, price, amount)
	if err != nil {
		s.logger.Error("[ProductService] Error of creating product", "name", name, "error", err)
		return 0, fmt.Errorf("product creation failed: %w", err)
	}

	// Публикуем событие о создании товара через KAFKA
	event := events.ProductCreated{
		ProductID: id,
		Name:      name,
		Price:     price,
		Amount:    amount,
	}
	if pubErr := s.publisher.Publish(ctx, "product-events", &event); pubErr != nil {
		s.logger.Warn("Failed to publish ProductCreated event", "error", pubErr)
		// Не возвращаем ошибку, чтобы не нарушать основную операцию
	}

	s.logger.Info("[ProductService] ✅ Product created successfully", "id", id, "name", name, "price", price, "amount", amount)
	return id, nil
}

func (s *ProductService) Get(id int64) (*models.Product, error) {
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

func (s *ProductService) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		s.logger.Warn("product id should be greater than zero")
		return errors.New("product id should be greater than zero")
	}

	tx, err := s.repo.NewTransaction(ctx)
	if err != nil {
		s.logger.Error("[ProductService] Error of creating transaction", "id", id, "error", err)
	}

	err = s.repo.DeleteProductWithTransaction(tx, id)
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
		tx.Commit()
		return nil
	}

	// Публикуем событие об удалении товара через KAFKA
	event := events.ProductDeleted{ProductID: id}
	pubErr := s.publisher.Publish(ctx, "product-events", &event)
	if pubErr != nil {
		s.logger.Warn("Failed to publish ProductDeleted event", "error", pubErr)
	}

	if price > 10000 {
		if pubErr != nil {
			tx.Rollback()
			return fmt.Errorf("product delete failed: %w", pubErr)
		}
	}

	s.logger.Info("[ProductService] ✅ Product deleted successfully", "id", id)
	tx.Commit()
	return nil
}

func (s *ProductService) AddProduct(id, amount int64) (int64, error) {
	if id <= 0 {
		s.logger.Warn("product id should be greater than zero")
		return 0, errors.New("product id should be greater than zero")
	}

	if amount <= 0 {
		s.logger.Warn("product amount should be greater than zero")
		return 0, errors.New("product amount should be greater than zero")
	}

	newAmount, err := s.repo.AddProduct(id, amount)
	if err != nil {
		s.logger.Error("[ProductService] Error of adding product", "id", id, "error", err)
		return 0, fmt.Errorf("product add failed: %w", err)
	}

	s.logger.Info("[ProductService] ✅ Product add successful", "id", id, "new amount", newAmount)
	return newAmount, nil
}
