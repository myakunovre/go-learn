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

type UserServiceInterface interface {
	Create(name, email, password string) (int, error)
}

type AuthService interface {
	Login(ctx context.Context, email, password string) (string, error)
	Authenticate(ctx context.Context, token string) (int, error)
	Logout(ctx context.Context, token string) error
}

type Handler struct {
	productService ProductServiceInterface
	orderService   OrderServiceInterface
	userService    UserServiceInterface
	authService    AuthService
	logger         *slog.Logger
}

func NewHandler(
	productService ProductServiceInterface,
	orderService OrderServiceInterface,
	userService UserServiceInterface,
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
