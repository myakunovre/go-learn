package repo

import (
	"database/sql"
	"fmt"
	"go-learn/models"
	"log"
)

type ProductRepository struct {
	db *sql.DB
}

func NewProductRepository(db *sql.DB) *ProductRepository {
	return &ProductRepository{db: db}
}

func (p *ProductRepository) DeleteProduct(id int) error {
	log.Printf("[ProductRepository] Deleting product with ID=%d", id)

	result, err := p.db.Exec("DELETE FROM products WHERE id = $1", id)
	if err != nil {
		log.Printf("[ProductRepository] Failed to delete product ID=%d: %v", id, err)
		return fmt.Errorf("failed to delete product: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("[ProductRepository] Failed to check rows affected for ID=%d: %v", id, err)
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		log.Printf("[ProductRepository] No product found with ID=%d", id)
		return fmt.Errorf("no product found with id %d", id)
	}

	log.Printf("[ProductRepository] Successfully deleted product ID=%d", id)
	return nil
}

func (p *ProductRepository) CreateProduct(name string, price float64) (int, error) {
	log.Printf("[ProductRepository] Creating product: name=%s, price=%.2f", name, price)

	var id int
	err := p.db.QueryRow(
		"INSERT INTO products (name, price) VALUES ($1, $2) RETURNING id", name, price,
	).Scan(&id)

	if err != nil {
		log.Printf("[ProductRepository] Failed to create product: %v", err)
		return 0, fmt.Errorf("failed to create product: %w", err)
	}

	log.Printf("[ProductRepository] Successfully created product with ID=%d", id)
	return id, nil
}

func (p *ProductRepository) GetProduct(id int) (*models.Product, error) {
	log.Printf("[ProductRepository] Getting product with ID=%d", id)

	var product models.Product
	err := p.db.QueryRow(
		"SELECT id, name, price FROM products WHERE id = $1", id,
	).Scan(&product.ID, &product.Name, &product.Price)

	if err != nil {
		log.Printf("[ProductRepository] Failed to get product with ID=%d: %v", id, err)
		return nil, fmt.Errorf("failed to get product: %w", err)
	}

	log.Printf("[ProductRepository] Successfully got product with ID=%d", id)
	return &product, nil
}

func (p *ProductRepository) GetAllProducts() ([]models.Product, error) {
	log.Printf("[ProductRepository] Getting all products")

	var products []models.Product
	rows, err := p.db.Query(
		"SELECT id, name, price FROM products")

	if err != nil {
		log.Printf("[ProductRepository] Failed to get all products: %v", err)
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
		log.Printf("[ProductRepository] Failed to get all products: %v", err)
		return nil, fmt.Errorf("error iterating products: %w", err)
	}

	if products == nil {
		products = []models.Product{}
	}

	log.Printf("[ProductRepository] Successfully got all %d products", len(products))
	return products, nil
}
