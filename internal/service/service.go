package service

import (
	"fmt"
	"go-learn/models"
	"log/slog"
)

type ProductRepository interface {
	DeleteProduct(int) error
	CreateProduct(name string, price float64) (int, error)
	GetProduct(int) (*models.Product, error)
	GetAllProducts() ([]models.Product, error)
}

type ProductService struct {
	repo   ProductRepository
	logger *slog.Logger
}

func NewProductService(repo ProductRepository, logger *slog.Logger) *ProductService {
	return &ProductService{
		repo:   repo,
		logger: logger,
	}
}

func (s *ProductService) Delete(id int) error {
	err := s.repo.DeleteProduct(id)
	if err != nil {
		s.logger.Error("[ProductService] Error of deleting product", "id", id, "error", err)
		return fmt.Errorf("failed to delete product with id %d: %v", id, err)
	}

	s.logger.Info("[ProductService] ✅ Product deleted successfully", "id", id)
	return nil
}

func (s *ProductService) Create(name string, price float64) (int, error) {
	id, err := s.repo.CreateProduct(name, price)
	if err != nil {
		s.logger.Error("[ProductService] Error of creating product", "id", id, "error", err)
		return 0, fmt.Errorf("product creation failed: %w", err)
	}

	s.logger.Info("[ProductService] ✅ Product created successful", "id", id, "name", name, "price", price)
	return id, nil
}

func (s *ProductService) Get(id int) (*models.Product, error) {
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
