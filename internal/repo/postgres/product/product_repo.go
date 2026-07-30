package product

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"go-learn/models"
	"log/slog"
)

type ProductRepository struct {
	db     *sql.DB
	logger *slog.Logger
}

func NewProductRepository(db *sql.DB, logger *slog.Logger) *ProductRepository {
	return &ProductRepository{
		db:     db,
		logger: logger,
	}
}

func (r *ProductRepository) CreateProduct(name string, price, amount int64) (int64, error) {
	r.logger.Debug("[ProductRepository] Creating product", "name", name, "price", price, "amount", amount)

	var id int64
	err := r.db.QueryRow(
		"INSERT INTO products (name, price, amount) VALUES ($1, $2, $3) RETURNING id", name, price, amount,
	).Scan(&id)

	if err != nil {
		r.logger.Error("[ProductRepository] Failed to create product", "err", err)
		return 0, fmt.Errorf("failed to create product: %w", err)
	}

	r.logger.Info("[ProductRepository] Product created successfully", "id", id, "name", name, "price", price, "amount", amount)
	return id, nil
}

func (r *ProductRepository) AddProduct(id, amount int64) (int64, error) {
	r.logger.Debug("[ProductRepository] Adding product", "id", id, "amount", amount)

	var curAmount int64
	err := r.db.QueryRow(
		"SELECT amount FROM products WHERE id = $1", id).Scan(&curAmount)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.logger.Warn("[ProductRepository] Product not found", "id", id)
			return 0, fmt.Errorf("product with id %d not found", id)
		}

		r.logger.Error("[ProductRepository] Failed to get product", "id", id, "err", err)
		return 0, fmt.Errorf("failed to get product: %w", err)
	}

	newAmount := curAmount + amount

	err = r.db.QueryRow(
		"UPDATE products SET amount = $1 WHERE id = $2", newAmount, id,
	).Scan(&newAmount)

	if err != nil {
		r.logger.Error("[ProductRepository] Failed to add product", "id", id, "err", err)
		return 0, fmt.Errorf("failed to add product: %w", err)
	}

	r.logger.Info("[ProductRepository] Product added successfully", "id", id, "amount", amount)
	return newAmount, nil
}

func (r *ProductRepository) GetProduct(id int64) (*models.Product, error) {
	r.logger.Debug("[ProductRepository] Getting product", "id", id)

	var product models.Product
	err := r.db.QueryRow(
		"SELECT id, name, price, amount FROM products WHERE id = $1", id,
	).Scan(&product.ID, &product.Name, &product.Price, &product.Amount)

	if err != nil {
		if err == sql.ErrNoRows {
			r.logger.Warn("[ProductRepository] Product not found", "id", id)
			return nil, fmt.Errorf("product with id %d not found", id)
		}

		r.logger.Error("[ProductRepository] Failed to get product", "id", id, "err", err)
		return nil, fmt.Errorf("failed to get product: %w", err)
	}

	r.logger.Info("[ProductRepository] Product found successfully", "id", id, "name", product.Name)
	return &product, nil
}

func (r *ProductRepository) GetAllProducts() ([]models.Product, error) {
	r.logger.Debug("[ProductRepository] Getting all products")

	rows, err := r.db.Query("SELECT id, name, price, amount FROM products")
	if err != nil {
		r.logger.Error("[ProductRepository] Failed to get all products", "error", err)
		return nil, fmt.Errorf("failed to query products: %w", err)
	}
	defer rows.Close()

	var products []models.Product
	for rows.Next() {
		var product models.Product
		if err := rows.Scan(&product.ID, &product.Name, &product.Price, &product.Amount); err != nil {
			r.logger.Error("[ProductRepository] Failed to scan product row", "error", err)
			return nil, fmt.Errorf("failed to scan product: %w", err)
		}
		products = append(products, product)
	}

	if err = rows.Err(); err != nil {
		r.logger.Error("[ProductRepository] Error iterating product rows", "error", err)
		return nil, fmt.Errorf("error iterating products: %w", err)
	}

	r.logger.Info("[ProductRepository] Products found successfully", "count", len(products))
	return products, nil
}

func (r *ProductRepository) DeleteProduct(id int64) error {
	r.logger.Debug("[ProductRepository] Deleting product]", "id", id)

	result, err := r.db.Exec("DELETE FROM products WHERE id = $1", id)
	if err != nil {
		r.logger.Error("[ProductRepository] Failed to delete product", "id", id, "err", err)
		return fmt.Errorf("failed to delete product: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		r.logger.Error("[ProductRepository] Failed to delete product", "id", id, "err", err)
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		r.logger.Warn("[ProductRepository] No product found to delete", "id", id)
		return fmt.Errorf("no product found with id %d", id)
	}

	r.logger.Info("[ProductRepository] Product deleted successfully", "id", id)
	return nil
}

func (r *ProductRepository) DeleteProductWithTransaction(tx *sql.Tx, id int64) error {
	r.logger.Debug("[ProductRepository] Deleting product]", "id", id)

	result, err := tx.Exec("DELETE FROM products WHERE id = $1", id)
	if err != nil {
		r.logger.Error("[ProductRepository] Failed to delete product", "id", id, "err", err)
		return fmt.Errorf("failed to delete product: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		r.logger.Error("[ProductRepository] Failed to delete product", "id", id, "err", err)
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		r.logger.Warn("[ProductRepository] No product found to delete", "id", id)
		return fmt.Errorf("no product found with id %d", id)
	}

	r.logger.Info("[ProductRepository] Product deleted successfully", "id", id)
	return nil
}

func (r *ProductRepository) NewTransaction(ctx context.Context) (*sql.Tx, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		r.logger.Error("failed to create transaction", "error", err)
	}
	return tx, err
}
