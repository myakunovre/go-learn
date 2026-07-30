package order

import (
	"database/sql"
	"errors"
	"fmt"
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

func (r *OrderRepository) CreateOrder(description string, userId int, products map[int]int) (int, error) {
	r.logger.Debug("[OrderRepository] Creating order", "description", description, "userId", userId)

	// Создаем транзакцию, т.к. добавляем данные в две таблицы
	tx, err := r.db.Begin()
	if err != nil {
		r.logger.Error("[OrderRepository] Failed to start transaction", "error", err)
		return 0, err
	}
	defer tx.Rollback()

	// добавление в таблицу orders
	var orderId int
	err = r.db.QueryRow(
		"INSERT INTO orders (description, user_id) VALUES ($1, $2) RETURNING id", description, userId,
	).Scan(&orderId)

	if err != nil {
		r.logger.Error("[OrderRepository] Failed to create order", "err", err)
		return 0, fmt.Errorf("failed to create order: %w", err)
	}

	// добавление в таблицу order_items
	for productId, amount := range products {

		// todo
		// Получаем информацию о товаре из core-сервиса (через gRPC)
		// Пока используем заглушку
		productName := fmt.Sprintf("product_%d", productId) // заглушка Name
		productPrice := 100                                 // заглушка Price

		_, err := tx.Exec(
			"INSERT INTO order_items (order_id, product_id, product_name, product_amount, product_price) VALUES ($1, $2, $3, $4, $5)",
			orderId, productId, productName, amount, productPrice,
		)
		if err != nil {
			r.logger.Error("[OrderRepository] Failed to create order_item", "productId", productId, "err", err)
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

func (r *OrderRepository) GetOrder(id int) (*models.Order, error) {
	r.logger.Debug("[OrderRepository] Getting order", "id", id)

	var order models.Order
	err := r.db.QueryRow(
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

func (r *OrderRepository) DeleteOrder(id int) error {
	r.logger.Debug("[OrderRepository] Deleting order", "id", id)

	result, err := r.db.Exec("DELETE FROM orders WHERE id = $1", id)
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

func (r *OrderRepository) MarkDeletedProduct(productId int) error {
	r.logger.Debug("[OrderRepository] Marking Deleted product", "productId", productId)

	_, err := r.db.Exec(
		"UPDATE order_items SET item_exists = false WHERE product_id = $1", productId,
	)
	if err != nil {
		r.logger.Error("[OrderRepository] Failed to mark product as deleted", "productId", productId)
		return fmt.Errorf("failed to mark product as deleted: %w", err)
	}

	r.logger.Info("[OrderRepository] Product marked as deleted", "productId", productId)
	return nil
}
