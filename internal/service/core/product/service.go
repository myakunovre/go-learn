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
	GetProduct(id int64) (*models.Product, error)
	GetProductsByIDs(ids []int64) ([]*models.Product, error)
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
		return 0, errors.New("core name is blank")
	}

	if amount <= 0 {
		s.logger.Error("[ProductService] Amount less than zero", "amount", amount)
		return 0, errors.New("amount less than zero")
	}

	id, err := s.repo.CreateProduct(name, price, amount)
	if err != nil {
		s.logger.Error("[ProductService] Error of creating core", "name", name, "error", err)
		return 0, fmt.Errorf("core creation failed: %w", err)
	}

	// Публикуем событие о создании товара через KAFKA
	event := events.ProductCreated{
		ProductID: id,
		Name:      name,
		Price:     price,
		Amount:    amount,
	}
	if pubErr := s.publisher.Publish(ctx, "core-events", &event); pubErr != nil {
		s.logger.Warn("Failed to publish ProductCreated event", "error", pubErr)
		// Не возвращаем ошибку, чтобы не нарушать основную операцию
	}

	s.logger.Info("[ProductService] ✅ Product created successfully", "id", id, "name", name, "price", price, "amount", amount)
	return id, nil
}

func (s *ProductService) Get(id int64) (*models.Product, error) {
	if id <= 0 {
		s.logger.Warn("core id should be greater than zero")
		return nil, errors.New("core id should be greater than zero")
	}

	product, err := s.repo.GetProduct(id)
	if err != nil {
		s.logger.Error("[ProductService] Error of getting core", "id", id, "error", err)
		return nil, fmt.Errorf("core get failed: %w", err)
	}

	s.logger.Info("[ProductService] ✅ Product got successful", "id", id, "name", product.Name)
	return product, nil
}

func (s *ProductService) GetProductsByIDs(ids []int64) ([]*models.Product, error) {
	if len(ids) == 0 {
		s.logger.Error("[ProductService] Products ids is empty")
		return nil, errors.New("products ids is empty")
	}

	products, err := s.repo.GetProductsByIDs(ids)
	if err != nil {
		s.logger.Error("[ProductService] Error of getting products", "id", ids, "error", err)
		return nil, fmt.Errorf("products get failed: %w", err)
	}

	s.logger.Info("[ProductService] ✅ Products got successful", "id", ids)
	return products, nil
}

func (s *ProductService) GetAllProducts() ([]models.Product, error) {
	products, err := s.repo.GetAllProducts()
	if err != nil {
		s.logger.Error("[ProductService] Error of getting all products", "error", err)
		return nil, fmt.Errorf("core get failed: %w", err)
	}

	s.logger.Info("[ProductService] All Products got successful", "num", len(products))
	return products, nil
}

func (s *ProductService) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		s.logger.Warn("core id should be greater than zero")
		return errors.New("core id should be greater than zero")
	}

	// Получаем продукт перед удалением
	product, err := s.repo.GetProduct(id)
	if err != nil {
		s.logger.Error("[ProductService] Error of getting core", "id", id, "error", err)
	}

	tx, err := s.repo.NewTransaction(ctx)
	if err != nil {
		s.logger.Error("[ProductService] Error of creating transaction", "id", id, "error", err)
	}

	err = s.repo.DeleteProductWithTransaction(tx, id)
	if err != nil {
		s.logger.Error("[ProductService] Error of deleting core", "id", id, "error", err)
		return fmt.Errorf("failed to delete core with id %d: %w", id, err)
	}

	// Публикуем событие для всех продуктов, кроме дешёвых
	if product.Price < 1000 {
		s.logger.Info("[ProductService] ✅ Product deleted successfully", "id", id)
		tx.Commit()
		return nil
	}

	// Публикуем событие об удалении товара через KAFKA
	event := events.ProductDeleted{ProductID: id}
	err = s.publisher.Publish(ctx, "core-events", &event)
	if err != nil {
		s.logger.Warn("Failed to publish ProductDeleted event", "error", err)
	}

	if product.Price > 10000 {
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("core delete failed: %w", err)
		}
	}

	s.logger.Info("[ProductService] ✅ Product deleted successfully", "id", id)
	tx.Commit()
	return nil
}

func (s *ProductService) AddProduct(id, amount int64) (int64, error) {
	if id <= 0 {
		s.logger.Warn("core id should be greater than zero")
		return 0, errors.New("core id should be greater than zero")
	}

	if amount <= 0 {
		s.logger.Warn("core amount should be greater than zero")
		return 0, errors.New("core amount should be greater than zero")
	}

	newAmount, err := s.repo.AddProduct(id, amount)
	if err != nil {
		s.logger.Error("[ProductService] Error of adding core", "id", id, "error", err)
		return 0, fmt.Errorf("core add failed: %w", err)
	}

	s.logger.Info("[ProductService] ✅ Product add successful", "id", id, "new amount", newAmount)
	return newAmount, nil
}
