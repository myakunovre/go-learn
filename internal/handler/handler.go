package handler

import (
	"context"
	"go-learn/models"
	"log/slog"
)

type ProductServiceInterface interface {
	Delete(id int) error
	Create(name string, price int) (int, error)
	Get(id int) (*models.Product, error)
	GetAllProducts() ([]models.Product, error)
}

type OrderServiceInterface interface {
	BuyProduct(ctx context.Context, productID int) (int64, error)
	GetOrderCount(ctx context.Context, productID int) (int64, error)
}

type Handler struct {
	//productService *service.ProductService
	productService ProductServiceInterface
	orderService   OrderServiceInterface
	logger         *slog.Logger
}

func NewHandler(productService ProductServiceInterface, orderService OrderServiceInterface, logger *slog.Logger) *Handler {
	return &Handler{productService: productService, orderService: orderService, logger: logger}
}
