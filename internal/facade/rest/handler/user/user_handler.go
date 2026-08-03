package user

import (
	"fmt"
	"go-learn/internal/facade/rest/handler/models/core"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

type UserService interface {
	Create(name, email, password string) (int, error)
}

type Handler struct {
	userService UserService
	logger      *slog.Logger
}

func NewHandler(userService UserService, logger *slog.Logger) *Handler {
	return &Handler{userService: userService, logger: logger}
}

func (h *Handler) CreateUser(c *gin.Context) {
	h.logger.Info("Called CreateUser")

	var req core.CreateUserRequest

	// Привязываем JSON к структуре
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Invalid request: %v", err),
		})
		h.logger.Warn("Invalid request", "error", err)
		return
	}

	// Проверка, что имя не пустое
	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "user name can not be empty",
		})
		h.logger.Warn("user name can not be empty")
		return
	}

	// Проверка, что пароль не пустой
	if req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "user password can not be empty",
		})
		h.logger.Warn("user password can not be empty")
		return
	}

	// Проверка, что имейл не пустой
	if req.Email == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "user email can not be empty",
		})
		h.logger.Warn("user email can not be empty")
		return
	}

	// Проверка, что пароль имеет не менее 8 символов
	if len(req.Password) < 8 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "user password length must be at least 8 symbols",
		})
		h.logger.Warn("user password length less than 8 symbols")
		return
	}

	// Создание пользователя через userService
	id, err := h.userService.Create(req.Name, req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		h.logger.Error("Error Create User", "name", req.Name, "email", req.Email)
		return
	}

	h.logger.Info("User created successfully", "id", id, "name", req.Name, "email", req.Email)
	c.JSON(http.StatusOK, gin.H{
		"message": "User created successfully",
		"id":      id,
		"name":    req.Name,
		"email":   req.Email,
	})
}
