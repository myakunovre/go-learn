package order

import (
	"context"
	"fmt"
	models2 "go-learn/internal/facade/rest/handler/models/order"
	order2 "go-learn/internal/service/order"
	"go-learn/models"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type OrderService interface {
	Create(ctx context.Context, input order2.CreateOrderInput) (int, error)
	Get(ctx context.Context, id int) (*models.Order, error)
	Delete(ctx context.Context, id int) error
	MarkDeletedProduct(ctx context.Context, productId int) error
}

type Handler struct {
	orderService OrderService
	logger       *slog.Logger
}

func NewHandler(orderService OrderService, logger *slog.Logger) *Handler {
	return &Handler{orderService: orderService, logger: logger}
}

func (h *Handler) CreateOrder(c *gin.Context) {
	h.logger.Info("Called CreateOrder")

	var req models2.CreateOrderRequest

	// Привязываем JSON к структуре
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Invalid request: %v", err),
		})
		h.logger.Warn("Invalid request", "error", err)
		return
	}

	if req.Description == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "order description can not be empty",
		})
		h.logger.Warn("order description can not be empty")
		return
	}

	if req.UserId <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "user id price can not be less than zero",
		})
		h.logger.Warn("user id can not be less than zero")
		return
	}

	if req.Products == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "products can not be empty",
		})
		h.logger.Warn("products can not be empty")
		return
	}

	for _, product := range req.Products {
		if product.ProductId <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "product id can not be less than zero",
			})
			h.logger.Warn("product id can not be less than zero")
			return
		}
		if product.Quantity <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "product quantity can not be less than zero",
			})
			h.logger.Warn("product quantity can not be less than zero")
			return
		}
	}

	products := make([]order2.OrderProduct, 0, len(req.Products))

	for _, product := range req.Products {
		products = append(products, order2.OrderProduct{
			ProductID: product.ProductId,
			Quantity:  product.Quantity,
		})
	}

	input := order2.CreateOrderInput{
		Description: req.Description,
		UserID:      req.UserId,
		Products:    products,
	}

	id, err := h.orderService.Create(c.Request.Context(), input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		h.logger.Error("Error Create Order", "description", req.Description, "error", err)
		return
	}

	h.logger.Info("Order created successfully", "id", id, "description", req.Description)
	c.JSON(http.StatusOK, gin.H{
		"message":     "Product created successfully",
		"id":          id,
		"description": req.Description,
		"userID":      req.UserId,
		"products":    req.Products,
	})
}

func (h *Handler) GetOrderById(c *gin.Context) {
	h.logger.Info("Called GetProductById", "id", c.Param("id"))

	// Получаем ID из URL‑пути
	idStr := c.Param("id")
	if idStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "ID parameter is required",
		})
		h.logger.Warn("ID parameter is required")
		return
	}

	// Конвертируем ID в число
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid id format",
		})
		h.logger.Warn("invalid id format")
		return
	}

	order, err := h.orderService.Get(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		h.logger.Error("Error Get Order", "id", id, "error", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"order": order,
	})

	h.logger.Info("Order found successfully", "description", order.Description, "id", id)
}

func (h *Handler) DeleteOrderById(c *gin.Context) {
	h.logger.Debug("Called DeleteOrderById with id=%d", c.Param("id"))

	// Получаем ID из URL‑пути
	idStr := c.Param("id")
	if idStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "ID parameter is required",
		})
		h.logger.Warn("ID parameter is required")
		return
	}

	// Конвертируем ID в число
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid id format",
		})
		h.logger.Warn("invalid id format")
		return
	}

	// Удаляем товар через сервис
	err = h.orderService.Delete(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		h.logger.Error("Error Deleting Order", "id", id, "error", err)
		return
	}

	h.logger.Info("Delete Order Success", "id", id)
	c.JSON(http.StatusOK, gin.H{
		"message": "Order deleted successfully",
	})
}
