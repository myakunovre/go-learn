package repo

import (
	"database/sql"
	"go-learn/models"
)

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

func (p *ProductRepository) CreateProduct(name string, price float64) (int, error) {
	var id int
	err := p.db.QueryRow(
		"INSERT INTO products (name, price) VALUES ($1, $2) RETURNING id", name, price,
	).Scan(&id)

	if err != nil {
		return 0, err
	}

	return id, nil
}

func (p *ProductRepository) GetProduct(id int) (*models.Product, error) {
	var product models.Product
	err := p.db.QueryRow(
		"SELECT id, name, price FROM products WHERE id = $1", id,
	).Scan(&product.ID, &product.Name, &product.Price)

	if err != nil {
		return nil, err
	}

	return &product, nil
}
