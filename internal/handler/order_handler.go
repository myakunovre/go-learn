package handler

import (
	"go-learn/internal/service"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type OrderHandler struct {
	svc *service.OrderService
}

func NewOrderHandler(svc *service.OrderService) *OrderHandler {
	return &OrderHandler{svc: svc}
}

func (h *OrderHandler) BuyProduct(c *gin.Context) {
	log.Printf("Called buy product with id=%s\n", c.Query("id"))
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

	// Записываем заказ в редис
	_, err = h.svc.BuyProduct(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		log.Printf("Error buy product with id=%d", id)
		return
	}

	c.Status(http.StatusOK)
}

func (h *OrderHandler) GetCounts(c *gin.Context) {
	log.Printf("Called get order counts for product with id=%s\n", c.Query("id"))
	idStr := c.Query("id")
	if idStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "ID parameter is required",
		})
		log.Println("ID parameter is required")
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid id format",
		})
		log.Println("invalid id format")
		return
	}

	orderCounts, err := h.svc.GetOrderCount(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		log.Printf("Error get order count for product with id=%d", id)
		return
	}

	c.String(http.StatusOK, "%d", orderCounts)
}
