package handler

// CreateProductRequest Структура для входящих данных продукта
type CreateProductRequest struct {
	Name  string `json:"name" binding:"required"`
	Price int    `json:"price" binding:"required"`
}
