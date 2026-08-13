package order

// CreateOrderRequest Структура для входящих данных заказа
type CreateOrderRequest struct {
	Description string         `json:"description" binding:"required"`
	UserId      int64          `json:"userId" binding:"required"`
	Products    []OrderProduct `json:"products" binding:"required"`
}

type OrderProduct struct {
	ProductId int64 `json:"productId" binding:"required"`
	Quantity  int   `json:"quantity" binding:"required"`
}
