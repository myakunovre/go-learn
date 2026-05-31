package service

import (
	"fmt"
	"go-learn/models"
)

type ProductRepository interface {
	DeleteProduct(int) error
	CreateProduct(name string, price float64) (int, error)
	GetProduct(int) (*models.Product, error)
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
		return fmt.Errorf("failed to delete product with id %d: %v", id, err)
	}
	return nil
}

func (s *ProductService) Create(name string, price float64) (int, error) {

	id, err := s.repo.CreateProduct(name, price)
	if err != nil {
		return 0, fmt.Errorf("product creation failed: %w", err)
	}
	return id, nil
}

func (s *ProductService) Get(id int) (*models.Product, error) {
	product, err := s.repo.GetProduct(id)
	if err != nil {
		return nil, fmt.Errorf("product get failed: %w", err)
	}
	return product, nil
}
