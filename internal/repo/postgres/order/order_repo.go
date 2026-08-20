package order

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"go-learn/internal/domain/order"
	"go-learn/models"
	"log/slog"
)

type OrderRepository struct {
	db     *sql.DB
	logger *slog.Logger
}

func NewOrderRepository(db *sql.DB, logger *slog.Logger) *OrderRepository {
	return &OrderRepository{
		db:     db,
		logger: logger,
	}
}

func (r *OrderRepository) CreateOrder(ctx context.Context, description string, userId int, products []order.Product) (int, error) {
	r.logger.Debug("[OrderRepository] Creating order", "description", description, "userId", userId)

	// Создаем транзакцию, т.к. добавляем данные в две таблицы
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
		ReadOnly:  false,
	})
	if err != nil {
		r.logger.Error("[OrderRepository] Failed to start transaction", "error", err)
		return 0, err
	}
	defer tx.Rollback()

	// добавление в таблицу orders
	var orderId int
	err = tx.QueryRowContext(ctx,
		"INSERT INTO orders (description, user_id) VALUES ($1, $2) RETURNING id", description, userId,
	).Scan(&orderId)
	if err != nil {
		r.logger.Error("[OrderRepository] Failed to create order", "err", err)
		return 0, fmt.Errorf("failed to create order: %w", err)
	}

	// todo: похоже на цикл в запросах БД
	// добавление в таблицу order_items
	for _, product := range products {
		_, err := tx.ExecContext(ctx,
			"INSERT INTO order_items (order_id, product_id, product_name, product_amount_in_core, product_amount_in_order, product_price) VALUES ($1, $2, $3, $4, $5, $6)",
			orderId, product.ProductId, product.ProductName, product.ProductAmountInCore, product.ProductAmountInOrder, product.ProductPrice,
		)
		if err != nil {
			r.logger.Error("[OrderRepository] Failed to create order_item", "productId", product.ProductId, "err", err)
			return 0, fmt.Errorf("failed to create order_item: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		r.logger.Error("[OrderRepository] Failed to commit transaction", "error", err)
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	r.logger.Info("[OrderRepository] Order created successfully", "id", orderId)
	return orderId, nil
}

func (r *OrderRepository) GetOrder(ctx context.Context, id int) (*models.Order, error) {
	r.logger.Debug("[OrderRepository] Getting order", "id", id)

	var order models.Order
	err := r.db.QueryRowContext(ctx,
		"SELECT id, description, uder_id FROM orders WHERE id = $1", id,
	).Scan(&order.ID, &order.Description, &order.UserId)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.logger.Warn("[OrderRepository] Order not found", "id", id)
			return nil, fmt.Errorf("order with id %d not found", id)
		}

		r.logger.Error("[OrderRepository] Failed to get order", "id", id, "err", err)
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	r.logger.Info("[OrderRepository] Order found successfully", "id", id, "description", order.Description, "user", order.UserId)
	return &order, nil
}

func (r *OrderRepository) DeleteOrder(ctx context.Context, id int) error {
	r.logger.Debug("[OrderRepository] Deleting order", "id", id)

	result, err := r.db.ExecContext(ctx,
		"DELETE FROM orders WHERE id = $1", id)
	if err != nil {
		r.logger.Error("[OrderRepository] Failed to delete order", "id", id, "err", err)
		return fmt.Errorf("failed to delete order: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		r.logger.Error("[OrderRepository] Failed to delete order", "id", id, "err", err)
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		r.logger.Warn("[OrderRepository] No order found to delete", "id", id)
		return fmt.Errorf("no order found with id %d", id)
	}

	r.logger.Info("[OrderRepository] Order deleted successfully", "id", id)
	return nil
}

func (r *OrderRepository) MarkDeletedProduct(ctx context.Context, productId int) error {
	r.logger.Debug("[OrderRepository] Marking Deleted core", "productId", productId)

	_, err := r.db.ExecContext(ctx,
		"UPDATE order_items SET item_exists = false WHERE product_id = $1", productId,
	)
	if err != nil {
		r.logger.Error("[OrderRepository] Failed to mark core as deleted", "productId", productId)
		return fmt.Errorf("failed to mark core as deleted: %w", err)
	}

	r.logger.Info("[OrderRepository] Product marked as deleted", "productId", productId)
	return nil
}
