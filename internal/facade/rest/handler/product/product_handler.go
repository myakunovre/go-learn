package product

import (
	"context"
	"fmt"
	models2 "go-learn/internal/facade/rest/handler/models"
	"go-learn/models"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type ProductService interface {
	Delete(ctx context.Context, id int64) error
	Create(ctx context.Context, name string, price, amount int64) (int64, error)
	Get(id int64) (*models.Product, error)
	GetAllProducts() ([]models.Product, error)
	AddProduct(id, amount int64) (int64, error)
}

type Handler struct {
	productService ProductService
	logger         *slog.Logger
}

func NewHandler(productService ProductService, logger *slog.Logger) *Handler {
	return &Handler{productService: productService, logger: logger}
}

func (h *Handler) CreateProduct(c *gin.Context) {
	h.logger.Info("Called CreateProduct")

	var req models2.CreateProductRequest

	// Привязываем JSON к структуре
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Invalid request: %v", err),
		})
		h.logger.Warn("Invalid request", "error", err)
		return
	}

	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "product name can not be empty",
		})
		h.logger.Warn("product name can not be empty")
		return
	}

	if req.Price <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "product price can not be less than zero",
		})
		h.logger.Warn("product price can not be less than zero")
		return
	}

	id, err := h.productService.Create(c.Request.Context(), req.Name, req.Price, req.Amount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		h.logger.Error("Error Create Product", "name", req.Name, "price", req.Price, "error", err)
		return
	}

	h.logger.Info("Product created successfully", "id", id, "name", req.Name, "price", req.Price)
	c.JSON(http.StatusOK, gin.H{
		"message": "Product created successfully",
		"id":      id,
		"name":    req.Name,
		"price":   req.Price,
	})
}

func (h *Handler) GetProductById(c *gin.Context) {
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

	product, err := h.productService.Get(int64(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		h.logger.Error("Error Get Product", "id", id, "error", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"product": product,
	})

	h.logger.Info("Product found successfully", "name", product.Name, "id", id)
}

func (h *Handler) GetAllProducts(c *gin.Context) {
	h.logger.Info("Called GetAllProducts")

	products, err := h.productService.GetAllProducts()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		h.logger.Error("Error Get All Products", "error", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"products": products,
	})
	h.logger.Info("Products found successfully", "products", len(products))
}

func (h *Handler) DeleteProductById(c *gin.Context) {
	h.logger.Debug("Called DeleteProductById with id=%d", c.Param("id"))

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
	err = h.productService.Delete(c.Request.Context(), int64(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		h.logger.Error("Error Deleting Product", "id", id, "error", err)
		return
	}

	h.logger.Info("Delete Product Success", "id", id)
	c.JSON(http.StatusOK, gin.H{
		"message": "Product deleted successfully",
	})
}

func (h *Handler) AddProduct(c *gin.Context) {
	h.logger.Info("Called AddProduct", "id", c.Param("id"), c.Param("amount"))

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

	// Получаем amount из URL‑пути
	amStr := c.Param("amount")
	if amStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Amount parameter is required",
		})
		h.logger.Warn("Amount parameter is required")
		return
	}

	// Конвертируем amount в число
	am, err := strconv.Atoi(amStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid amount format",
		})
		h.logger.Warn("invalid amount format")
		return
	}

	amount, err := h.productService.AddProduct(int64(id), int64(am))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		h.logger.Error("Error Add Product", "id", id, "error", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"amount": amount,
	})

	h.logger.Info("Product add successfully", "amount", amount, "id", id)
}
