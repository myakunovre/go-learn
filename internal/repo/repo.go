package repo

import (
	"database/sql"
	"fmt"
	"go-learn/models"
)

type ProductRepository struct {
	db *sql.DB
}

func NewProductRepository(db *sql.DB) *ProductRepository {
	return &ProductRepository{db: db}
}

func (p *ProductRepository) DeleteProduct(id int) error {
	result, err := p.db.Exec("DELETE FROM products WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("failed to delete product: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no product found with id %d", id)
	}

	return nil
}

func (p *ProductRepository) CreateProduct(name string, price float64) (int, error) {
	var id int
	err := p.db.QueryRow(
		"INSERT INTO products (name, price) VALUES ($1, $2) RETURNING id", name, price,
	).Scan(&id)

	if err != nil {
		return 0, fmt.Errorf("failed to create product: %w", err)
	}

	return id, nil
}

func (p *ProductRepository) GetProduct(id int) (*models.Product, error) {
	var product models.Product
	err := p.db.QueryRow(
		"SELECT id, name, price FROM products WHERE id = $1", id,
	).Scan(&product.ID, &product.Name, &product.Price)

	if err != nil {
		return nil, fmt.Errorf("failed to get product: %w", err)
	}

	return &product, nil
}

func (p *ProductRepository) GetAllProducts() ([]models.Product, error) {
	var products []models.Product
	rows, err := p.db.Query(
		"SELECT id, name, price FROM products")
	if err != nil {
		return nil, fmt.Errorf("failed to query products: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var product models.Product
		if err := rows.Scan(&product.ID, &product.Name, &product.Price); err != nil {
			return nil, fmt.Errorf("failed to scan product: %w", err)
		}
		products = append(products, product)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating products: %w", err)
	}

	if products == nil {
		products = []models.Product{}
	}

	return products, nil
}
