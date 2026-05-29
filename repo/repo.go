package repo

import "database/sql"

type ProductRepository struct {
	db *sql.DB
}

func NewProductRepository(db *sql.DB) *ProductRepository {
	return &ProductRepository{db: db}
}

func (p *ProductRepository) DeleteProduct(id int) error {
	_, err := p.db.Exec("DELETE FROM products WHERE id = $1", id)
	return err
}
