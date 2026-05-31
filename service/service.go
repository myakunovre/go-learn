package service

type ProductRepository interface {
	DeleteProduct(int) error
}

type ProductService struct {
	repo ProductRepository
}

func NewProductService(repo ProductRepository) *ProductService {
	return &ProductService{repo: repo}
}

func (s *ProductService) Delete(id int) error {
	err := s.repo.DeleteProduct(id)
	return err
}
