package core

// CreateProductRequest Структура для входящих данных продукта
type CreateProductRequest struct {
	Name   string `json:"name" binding:"required"`
	Price  int64  `json:"price" binding:"required"`
	Amount int64  `json:"amount" binding:"required"`
}

// CreateUserRequest Структура для входящих данных пользователя
type CreateUserRequest struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}
