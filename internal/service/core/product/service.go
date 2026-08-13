package product

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"go-learn/internal/events"
	"go-learn/models"
	"log/slog"
	"time"
)

type ProductRepository interface {
	CreateProduct(ctx context.Context, name string, price, amount int64) (int64, error)
	AddProduct(ctx context.Context, id, amount int64) (int64, error)
	GetProduct(ctx context.Context, id int64) (*models.Product, error)
	GetProductsByIDs(ctx context.Context, ids []int64) ([]*models.Product, error)
	GetAllProducts(ctx context.Context) ([]models.Product, error)
	DeleteProduct(ctx context.Context, id int64) error
	DeleteProductWithTransaction(ctx context.Context, tx *sql.Tx, id int64) error
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

	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	id, err := s.repo.CreateProduct(dbCtx, name, price, amount)
	if err != nil {
		s.logger.Error("[ProductService] Error of creating core", "name", name, "error", err)
		return 0, fmt.Errorf("core creation failed: %w", err)
	}

	// Публикуем событие о создании товара через KAFKA
	eventCtx, eventCancel := context.WithTimeout(ctx, 2*time.Second)
	defer eventCancel()

	event := events.ProductCreated{
		ProductID: id,
		Name:      name,
		Price:     price,
		Amount:    amount,
	}
	if pubErr := s.publisher.Publish(eventCtx, "core-events", &event); pubErr != nil {
		s.logger.Warn("Failed to publish ProductCreated event", "error", pubErr)
		// Не возвращаем ошибку, чтобы не нарушать основную операцию
	}

	s.logger.Info("[ProductService] ✅ Product created successfully", "id", id, "name", name, "price", price, "amount", amount)
	return id, nil
}

func (s *ProductService) Get(ctx context.Context, id int64) (*models.Product, error) {
	if id <= 0 {
		s.logger.Warn("core id should be greater than zero")
		return nil, errors.New("core id should be greater than zero")
	}

	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	product, err := s.repo.GetProduct(dbCtx, id)
	if err != nil {
		s.logger.Error("[ProductService] Error of getting core", "id", id, "error", err)
		return nil, fmt.Errorf("core get failed: %w", err)
	}

	s.logger.Info("[ProductService] ✅ Product got successful", "id", id, "name", product.Name)
	return product, nil
}

func (s *ProductService) GetProductsByIDs(ctx context.Context, ids []int64) ([]*models.Product, error) {
	if len(ids) == 0 {
		s.logger.Error("[ProductService] Products ids is empty")
		return nil, errors.New("products ids is empty")
	}

	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	products, err := s.repo.GetProductsByIDs(dbCtx, ids)
	if err != nil {
		s.logger.Error("[ProductService] Error of getting products", "id", ids, "error", err)
		return nil, fmt.Errorf("products get failed: %w", err)
	}

	s.logger.Info("[ProductService] ✅ Products got successful", "id", ids)
	return products, nil
}

func (s *ProductService) GetAllProducts(ctx context.Context) ([]models.Product, error) {

	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	products, err := s.repo.GetAllProducts(dbCtx)
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

	deleteCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Получаем продукт перед удалением
	product, err := s.repo.GetProduct(deleteCtx, id)
	if err != nil {
		s.logger.Error("[ProductService] Error of getting core", "id", id, "error", err)
		return fmt.Errorf("core delete product failed: %w", err)
	}

	tx, err := s.repo.NewTransaction(deleteCtx)
	if err != nil {
		s.logger.Error("[ProductService] Error of creating transaction", "id", id, "error", err)
		return fmt.Errorf("failed to create transaction: %w", err)
	}

	err = s.repo.DeleteProductWithTransaction(deleteCtx, tx, id)
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
	eventCtx, eventCancel := context.WithTimeout(ctx, 2*time.Second)
	defer eventCancel()

	event := events.ProductDeleted{ProductID: id}
	err = s.publisher.Publish(eventCtx, "core-events", &event)
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
	if err := tx.Commit(); err != nil {
		s.logger.Error("[ProductService] Error of committing transaction", "id", id, "error", err)
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (s *ProductService) AddProduct(ctx context.Context, id, amount int64) (int64, error) {
	if id <= 0 {
		s.logger.Warn("core id should be greater than zero")
		return 0, errors.New("core id should be greater than zero")
	}

	if amount <= 0 {
		s.logger.Warn("core amount should be greater than zero")
		return 0, errors.New("core amount should be greater than zero")
	}

	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	newAmount, err := s.repo.AddProduct(dbCtx, id, amount)
	if err != nil {
		s.logger.Error("[ProductService] Error of adding core", "id", id, "error", err)
		return 0, fmt.Errorf("core add failed: %w", err)
	}

	s.logger.Info("[ProductService] ✅ Product add successful", "id", id, "new amount", newAmount)
	return newAmount, nil
}
