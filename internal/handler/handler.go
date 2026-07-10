package handler

import (
	"context"
	"go-learn/models"
	"log/slog"
)

type ProductService interface {
	Delete(id int) error
	Create(name string, price int) (int, error)
	Get(id int) (*models.Product, error)
	GetAllProducts() ([]models.Product, error)
}

type OrderService interface {
	BuyProduct(ctx context.Context, productID int) (int64, error)
	GetOrderCount(ctx context.Context, productID int) (int64, error)
}

type UserService interface {
	Create(name, email, password string) (int, error)
}

type AuthService interface {
	Login(ctx context.Context, email, password string) (string, error)
	Authenticate(ctx context.Context, token string) (int, error)
	Logout(ctx context.Context, token string) error
}

type Handler struct {
	productService ProductService
	orderService   OrderService
	userService    UserService
	authService    AuthService
	logger         *slog.Logger
}

func NewHandler(
	productService ProductService,
	orderService OrderService,
	userService UserService,
	authService AuthService,
	logger *slog.Logger) *Handler {

	return &Handler{
		productService: productService,
		orderService:   orderService,
		userService:    userService,
		authService:    authService,
		logger:         logger,
	}
}
