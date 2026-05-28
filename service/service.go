package service

type DB interface {
	DeleteProduct(int) error
}

type Service struct {
	db DB
}

func NewService(db DB) *Service {
	return &Service{db: db}
}

func (s *Service) Delete(id int) error {
	err := s.db.DeleteProduct(id)
	return err
}
