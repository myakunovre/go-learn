package main

import (
	"database/sql"
	"fmt"
	"go-learn/internal/handler"
	"go-learn/internal/repo"
	"go-learn/internal/service"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq" // Драйвер для Postgres
)

func main() {

	err := godotenv.Load()
	if err != nil {
		fmt.Println("⚠️  .env файл не найден, использую системные переменные")
	}

	dbUser := os.Getenv("DB_USER")
	if dbUser == "" {
		dbUser = "postgres"
	}

	dbPass := os.Getenv("DB_PASSWORD")
	if dbPass == "" {
		dbPass = "1234"
	}

	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}

	dbPort := os.Getenv("DB_PORT")
	if dbPort == "" {
		dbPort = "5432"
	}

	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		dbName = "postgres"
	}

	serverPort := os.Getenv("SERVER_PORT")
	if serverPort == "" {
		serverPort = "8080"
	}

	conStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPass, dbName,
	)

	// === ПОДКЛЮЧЕНИЕ К БАЗЕ ДАННЫХ ===
	// Подключение к БД
	db, err := sql.Open("postgres", conStr)
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
	handlers := handler.NewHandler(productService)

	router := gin.Default()

	router.DELETE("/product/delete", handlers.DeleteProductById)
	router.POST("/product/create", handlers.CreateProduct)
	router.GET("/product/:id", handlers.GetProductById)
	router.GET("/products", handlers.GetAllProducts)

	// Запуск http-сервиса
	fmt.Println("🌐 Started to http://localhost:" + serverPort)
	err = router.Run(":" + serverPort)
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
