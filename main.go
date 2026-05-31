package main

import (
	"database/sql"
	"fmt"
	"go-learn/repo"
	"go-learn/service"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq" // Драйвер для Postgres
)

func main() {
	// === ПОДКЛЮЧЕНИЕ К БАЗЕ ДАННЫХ ===
	// Строка подключения к Postgres
	connStr := "host=localhost port=5432 user=user password=password dbname=products_db sslmode=disable"

	// Подключение к БД
	db, err := sql.Open("postgres", connStr)
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
	// Rout for deleteProductHandler product
	router := gin.Default()
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
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Product deleted successfully",
		})
	})

	// Запуск http-сервиса
	fmt.Println("🌐 Started to http://localhost:8080")
	err = router.Run(":8080")
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
