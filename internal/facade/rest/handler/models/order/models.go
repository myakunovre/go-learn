package order

import "go-learn/internal/service/order"

// CreateOrderRequest Структура для входящих данных заказа
type CreateOrderRequest struct {
	Description string               `json:"description" binding:"required"`
	UserId      int64                `json:"userId" binding:"required"`
	Products    []order.OrderProduct `json:"products" binding:"required"`
}
