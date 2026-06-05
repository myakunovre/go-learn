package service

import (
	"fmt"
	"go-learn/models"
	"log"
)

type ProductRepository interface {
	DeleteProduct(int) error
	CreateProduct(name string, price float64) (int, error)
	GetProduct(int) (*models.Product, error)
	GetAllProducts() ([]models.Product, error)
}

type ProductService struct {
	repo ProductRepository
}

func NewProductService(repo ProductRepository) *ProductService {
	return &ProductService{repo: repo}
}

func (s *ProductService) Delete(id int) error {
	err := s.repo.DeleteProduct(id)
	if err != nil {
		log.Printf("[ProductService] Error of deleting product with ID=%d: %v", id, err)
		return fmt.Errorf("failed to delete product with id %d: %v", id, err)
	}
	log.Printf("[ProductService] ✅ Product with ID=%d deleted successful", id)
	return nil
}

func (s *ProductService) Create(name string, price float64) (int, error) {

	id, err := s.repo.CreateProduct(name, price)
	if err != nil {
		log.Printf("[ProductService] Error of creation product with ID=%d: %v", id, err)
		return 0, fmt.Errorf("product creation failed: %w", err)
	}
	log.Printf("[ProductService] ✅ Product with ID=%d created successful", id)
	return id, nil
}

func (s *ProductService) Get(id int) (*models.Product, error) {
	product, err := s.repo.GetProduct(id)
	if err != nil {
		log.Printf("[ProductService] Error of getting product with ID=%d: %v", id, err)
		return nil, fmt.Errorf("product get failed: %w", err)
	}
	log.Printf("[ProductService] ✅ Product with ID=%d got successful", id)
	return product, nil
}

func (s *ProductService) GetAllProducts() ([]models.Product, error) {
	products, err := s.repo.GetAllProducts()
	if err != nil {
		log.Printf("[ProductService] Error of getting all products: %v", err)
		return nil, fmt.Errorf("product get failed: %w", err)
	}
	log.Println("[ProductService] ✅ All product with got successful")
	return products, nil
}
