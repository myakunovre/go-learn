package handler

// CreateProductRequest Структура для входящих данных продукта
type CreateProductRequest struct {
	Name  string  `json:"name" binding:"required"`
	Price float64 `json:"price" binding:"required"`
}
