package service

type DB interface {
	DeleteProduct(int) error
}

type ProductService struct {
	db DB
}

func NewProductService(db DB) *ProductService {
	return &ProductService{db: db}
}

func (s *ProductService) Delete(id int) error {
	err := s.db.DeleteProduct(id)
	return err
}
