package product

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"go-learn/models"
	"log/slog"
	"strings"
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

func (r *ProductRepository) CreateProduct(ctx context.Context, name string, price, amount int64) (int64, error) {
	r.logger.Debug("[ProductRepository] Creating core", "name", name, "price", price, "amount", amount)

	var id int64
	err := r.db.QueryRowContext(
		ctx,
		"INSERT INTO products (name, price, amount) VALUES ($1, $2, $3) RETURNING id", name, price, amount,
	).Scan(&id)

	if err != nil {
		r.logger.Error("[ProductRepository] Failed to create core", "err", err)
		return 0, fmt.Errorf("failed to create core: %w", err)
	}

	r.logger.Info("[ProductRepository] Product created successfully", "id", id, "name", name, "price", price, "amount", amount)
	return id, nil
}

func (r *ProductRepository) AddProduct(ctx context.Context, id, amount int64) (int64, error) {
	r.logger.Debug("[ProductRepository] Adding core", "id", id, "amount", amount)

	var curAmount int64
	err := r.db.QueryRowContext(
		ctx,
		"SELECT amount FROM products WHERE id = $1", id).Scan(&curAmount)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.logger.Warn("[ProductRepository] Product not found", "id", id)
			return 0, fmt.Errorf("core with id %d not found", id)
		}

		r.logger.Error("[ProductRepository] Failed to get core", "id", id, "err", err)
		return 0, fmt.Errorf("failed to get core: %w", err)
	}

	newAmount := curAmount + amount

	err = r.db.QueryRowContext(ctx,
		"UPDATE products SET amount = $1 WHERE id = $2", newAmount, id,
	).Scan(&newAmount)

	if err != nil {
		r.logger.Error("[ProductRepository] Failed to add core", "id", id, "err", err)
		return 0, fmt.Errorf("failed to add core: %w", err)
	}

	r.logger.Info("[ProductRepository] Product added successfully", "id", id, "amount", amount)
	return newAmount, nil
}

func (r *ProductRepository) GetProduct(ctx context.Context, id int64) (*models.Product, error) {
	r.logger.Debug("[ProductRepository] Getting core", "id", id)

	var product models.Product
	err := r.db.QueryRowContext(
		ctx,
		"SELECT id, name, price, amount FROM products WHERE id = $1", id,
	).Scan(&product.ID, &product.Name, &product.Price, &product.Amount)

	if err != nil {
		if err == sql.ErrNoRows {
			r.logger.Warn("[ProductRepository] Product not found", "id", id)
			return nil, fmt.Errorf("core with id %d not found", id)
		}

		r.logger.Error("[ProductRepository] Failed to get core", "id", id, "err", err)
		return nil, fmt.Errorf("failed to get core: %w", err)
	}

	r.logger.Info("[ProductRepository] Product found successfully", "id", id, "name", product.Name)
	return &product, nil
}

func (r *ProductRepository) GetProductsByIDs(ctx context.Context, ids []int64) ([]*models.Product, error) {
	r.logger.Debug("[ProductRepository] Getting products", "id", ids)

	if len(ids) == 0 {
		r.logger.Debug("[ProductRepository] No products found")
		return []*models.Product{}, nil
	}

	// Создаем плейсхолдеры для каждого ID
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	// Формируем финальный запрос
	query := fmt.Sprintf("SELECT * FROM products WHERE id IN (%s)", strings.Join(placeholders, ","))

	// Выполняем запрос с параметрами
	row, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		r.logger.Error("[ProductRepository] Failed to query products", "id", ids, "err", err)
		return nil, fmt.Errorf("failed to query products: %w", err)
	}
	defer row.Close()

	// Сканируем результаты
	var products []*models.Product
	for row.Next() {
		var product models.Product
		err = row.Scan(&product.ID, &product.Name, &product.Price, &product.Amount)
		if err != nil {
			r.logger.Error("[ProductRepository] Failed to scan core", "id", ids, "err", err)
			return nil, fmt.Errorf("failed to scan core: %w", err)
		}
		products = append(products, &product)
	}

	r.logger.Info("[ProductRepository] Products found successfully", "id", ids)
	return products, nil

}

func (r *ProductRepository) GetAllProducts(ctx context.Context) ([]models.Product, error) {
	r.logger.Debug("[ProductRepository] Getting all products")

	rows, err := r.db.QueryContext(ctx, "SELECT id, name, price, amount FROM products")
	if err != nil {
		r.logger.Error("[ProductRepository] Failed to get all products", "error", err)
		return nil, fmt.Errorf("failed to query products: %w", err)
	}
	defer rows.Close()

	var products []models.Product
	for rows.Next() {
		var product models.Product
		if err := rows.Scan(&product.ID, &product.Name, &product.Price, &product.Amount); err != nil {
			r.logger.Error("[ProductRepository] Failed to scan core row", "error", err)
			return nil, fmt.Errorf("failed to scan core: %w", err)
		}
		products = append(products, product)
	}

	if err = rows.Err(); err != nil {
		r.logger.Error("[ProductRepository] Error iterating core rows", "error", err)
		return nil, fmt.Errorf("error iterating products: %w", err)
	}

	r.logger.Info("[ProductRepository] Products found successfully", "count", len(products))
	return products, nil
}

func (r *ProductRepository) DeleteProduct(ctx context.Context, id int64) error {
	r.logger.Debug("[ProductRepository] Deleting core]", "id", id)

	result, err := r.db.ExecContext(ctx, "DELETE FROM products WHERE id = $1", id)
	if err != nil {
		r.logger.Error("[ProductRepository] Failed to delete core", "id", id, "err", err)
		return fmt.Errorf("failed to delete core: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		r.logger.Error("[ProductRepository] Failed to delete core", "id", id, "err", err)
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		r.logger.Warn("[ProductRepository] No core found to delete", "id", id)
		return fmt.Errorf("no core found with id %d", id)
	}

	r.logger.Info("[ProductRepository] Product deleted successfully", "id", id)
	return nil
}

func (r *ProductRepository) DeleteProductWithTransaction(ctx context.Context, tx *sql.Tx, id int64) error {
	r.logger.Debug("[ProductRepository] Deleting core]", "id", id)

	result, err := tx.ExecContext(ctx, "DELETE FROM products WHERE id = $1", id)
	if err != nil {
		r.logger.Error("[ProductRepository] Failed to delete core", "id", id, "err", err)
		return fmt.Errorf("failed to delete core: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		r.logger.Error("[ProductRepository] Failed to delete core", "id", id, "err", err)
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		r.logger.Warn("[ProductRepository] No core found to delete", "id", id)
		return fmt.Errorf("no core found with id %d", id)
	}

	r.logger.Info("[ProductRepository] Product deleted successfully", "id", id)
	return nil
}

func (r *ProductRepository) NewTransaction(ctx context.Context) (*sql.Tx, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
		ReadOnly:  false,
	})
	if err != nil {
		r.logger.Error("failed to create transaction", "error", err)
	}
	return tx, err
}
