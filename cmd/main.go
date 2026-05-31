package main

import (
	"database/sql"
	"fmt"
	"go-learn/config"
	"go-learn/internal/repo"
	"go-learn/internal/service"
	"go-learn/models"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq" // Драйвер для Postgres
)

func main() {
	// === ПОДКЛЮЧЕНИЕ К БАЗЕ ДАННЫХ ===

	// Подключение к БД
	db, err := sql.Open("postgres", config.Config.DatabaseURL)
	if err != nil {
		log.Fatalf("Ошибка подключения к базе данных: %v", err)
	}
	defer db.Close() // Закроем соединение когда программа завершится

	// Проверка соединения
	err = db.Ping()
	if err != nil {
		log.Fatalf("Не удалось подключиться к Postgres: %v", err)
	}
	fmt.Println("✅ Успешно подключено к Postgres!!!")

	// === СОЗДАЕМ ТАБЛИЦУ ТОВАРОВ ===
	createTableProducts(db)

	productRepository := repo.NewProductRepository(db)
	productService := service.NewProductService(productRepository)

	// === НАСТРАИВАЕМ GIN-ROUTER ===
	router := gin.Default()

	// Rout for deleting product
	router.DELETE("/product/delete", func(c *gin.Context) {
		// Получаем ID из query параметра
		idStr := c.Query("id")
		if idStr == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "ID parameter is required",
			})
			return
		}

		// Конвертируем ID в число
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid id format",
			})
			return
		}

		// Удаляем товар через сервис
		err = productService.Delete(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Product deleted successfully",
		})
	})

	// Rout for creating product
	router.POST("/product/create", func(c *gin.Context) {
		var req models.CreateProductRequest

		// Привязываем JSON к структуре
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("Invalid request: %v", err),
			})
			return
		}

		if req.Name == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "product name can not be empty",
			})
			return
		}

		if req.Price <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "product price can not be less than zero",
			})
			return
		}

		id, err := productService.Create(req.Name, req.Price)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Product created successfully",
			"id":      id,
			"name":    req.Name,
			"price":   req.Price,
		})
	})

	router.GET("/product/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)

		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid id format",
			})
			return
		}

		product, err := productService.Get(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"product": product,
		})
	})

	// Запуск http-сервиса
	fmt.Println("🌐 Started to http://localhost:8080")
	err = router.Run(":" + config.Config.ServerPort)
	if err != nil {
		log.Fatalf("Ошибка запуска сервера: %v", err)
	}
}

func createTableProducts(db *sql.DB) {
	query := `
	CREATE TABLE IF NOT EXISTS products (
		id SERIAL PRIMARY KEY,
		name VARCHAR(100) NOT NULL,
		price NUMERIC(10, 2) NOT NULL
	);`

	_, err := db.Exec(query)
	if err != nil {
		log.Fatalf("Ошибка создания таблицы: %v", err)
	}

	fmt.Println("✅ Таблица 'products' проверена/создана успешно.")
}
