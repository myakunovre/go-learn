package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func (h *Handler) BuyProduct(c *gin.Context) {
	h.logger.Debug("[OrderHandler] Buying product", "productId", c.Param("id"))

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

	// Записываем заказ в редис
	_, err = h.orderService.BuyProduct(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		h.logger.Error("Error Buy Product", "id", id, "error", err)
		return
	}

	h.logger.Info("Success buy product", "id", id)
	c.Status(http.StatusOK)
}

func (h *Handler) GetCounts(c *gin.Context) {
	h.logger.Debug("Called get order counts for product", "id", c.Param("id"))

	// Получаем ID из URL‑пути
	idStr := c.Param("id")
	if idStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "ID parameter is required",
		})

		h.logger.Warn("ID parameter is required")
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid id format",
		})

		h.logger.Warn("invalid id format")
		return
	}

	orderCounts, err := h.orderService.GetOrderCount(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		h.logger.Error("Error Get Order Count for Product", "id", id, "error", err)
		return
	}

	h.logger.Info("Success Get Order Count for Product", "id", id)
	c.String(http.StatusOK, "%d", orderCounts)
}
