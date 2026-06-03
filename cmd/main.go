package main

import (
	"context"
	_ "context"
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
	"github.com/redis/go-redis/v9"
	_ "github.com/redis/go-redis/v9"
)

func main() {

	err := godotenv.Load()
	if err != nil {
		fmt.Println("⚠️  .env файл не найден, использую системные переменные")
	}

	// === ЗАГРУЗКА ПЕРЕМЕННЫХ ===
	dbUser := getEnv("DB_USER", "postgres")
	dbPass := getEnv("DB_PASSWORD", "1234")
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbName := getEnv("DB_NAME", "postgres")
	serverPort := getEnv("SERVER_PORT", "8080")
	redisHost := getEnv("REDIS_HOST", "localhost")
	redisPort := getEnv("REDIS_PORT", "6379")
	redisPassword := getEnv("REDIS_PASSWORD", "")

	// === ПОДКЛЮЧЕНИЕ К БАЗЕ ДАННЫХ POSTGRES ===
	conStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPass, dbName,
	)

	db, err := sql.Open("postgres", conStr)
	if err != nil {
		log.Fatalf("Ошибка подключения к базе данных: %v", err)
	}
	defer db.Close() // Закроем соединение когда программа завершится

	err = db.Ping()
	if err != nil {
		log.Fatalf("Не удалось подключиться к Postgres: %v", err)
	}
	fmt.Println("✅ Успешно подключено к Postgres!")

	// === ПОДКЛЮЧЕНИЕ К REDIS===
	redisClient := redis.NewClient(&redis.Options{
		Addr:     redisHost + ":" + redisPort,
		Password: redisPassword,
		DB:       0,
	})

	// Проверка соединения с Redis
	ctx := context.Background()
	err = redisClient.Ping(ctx).Err()
	if err != nil {
		log.Fatalf("Не удалось подключиться к Redis: %v", err)
	}
	fmt.Println("✅ Успешно подключено к Redis!")

	// === СОЗДАЕМ ТАБЛИЦУ ТОВАРОВ ===
	createTableProducts(db)

	// === ИНИЦИАЛИЗАЦИЯ СЛОЕВ ===
	productRepository := repo.NewProductRepository(db)
	productService := service.NewProductService(productRepository)
	productHandler := handler.NewHandler(productService)

	orderRepository := repo.NewOrderRepository(redisClient)
	orderService := service.NewOrderService(orderRepository)
	orderHandler := handler.NewOrderHandler(orderService)

	// === НАСТРОЙКА РОУТЕРА ===
	router := gin.Default()

	// Эндпоинты товаров
	router.DELETE("/product/delete", productHandler.DeleteProductById)
	router.POST("/product/create", productHandler.CreateProduct)
	router.GET("/product/:id", productHandler.GetProductById)
	router.GET("/products", productHandler.GetAllProducts)

	// Эндпоинты продуктов
	router.POST("/buy/product", orderHandler.BuyProduct)
	router.GET("/product/orders", orderHandler.GetCounts)

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

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
