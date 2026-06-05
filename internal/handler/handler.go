package handler

import (
	"fmt"
	"go-learn/internal/service"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *service.ProductService
}

func NewHandler(svc *service.ProductService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) DeleteProductById(c *gin.Context) {
	log.Printf("Called DeleteProductById with id=%d", c.Param("id"))

	// Получаем ID из query параметра
	idStr := c.Query("id")
	if idStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "ID parameter is required",
		})
		log.Println("ID parameter is required")
		return
	}

	// Конвертируем ID в число
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid id format",
		})
		log.Println("invalid id format")
		return
	}

	// Удаляем товар через сервис
	err = h.svc.Delete(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		log.Printf("Error Deleting Product with ID=%d: %v", id, err)
		return
	}

	log.Println("Delete Product Success")
	c.JSON(http.StatusOK, gin.H{
		"message": "Product deleted successfully",
	})
}

func (h *Handler) CreateProduct(c *gin.Context) {
	log.Println("Called CreateProduct")

	var req CreateProductRequest

	// Привязываем JSON к структуре
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Invalid request: %v", err),
		})
		log.Println("Invalid request")
		return
	}

	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "product name can not be empty",
		})
		log.Println("product name can not be empty")
		return
	}

	if req.Price <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "product price can not be less than zero",
		})
		log.Println("product price can not be less than zero")
		return
	}

	id, err := h.svc.Create(req.Name, req.Price)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		log.Printf("Error Create Product with id=%d: %v", id, err)
		return
	}

	log.Printf("Product named \"%s\" with price %f was created with id=%d", req.Name, req.Price, id)
	c.JSON(http.StatusOK, gin.H{
		"message": "Product created successfully",
		"id":      id,
		"name":    req.Name,
		"price":   req.Price,
	})
}

func (h *Handler) GetProductById(c *gin.Context) {
	log.Printf("Called GetProductById with id=%d", c.Param("id"))

	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid id format",
		})
		log.Println("invalid id format")
		return
	}

	product, err := h.svc.Get(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		log.Printf("Error of Get Product with id=%d: %v", id, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"product": product,
	})
	log.Printf("Product named \"%s\" with id=%d called successful", product.Name, id)
}

func (h *Handler) GetAllProducts(c *gin.Context) {
	products, err := h.svc.GetAllProducts()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		log.Println("GetAll Products Error")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"products": products,
	})
	log.Println("GetAll Products Success")
}
