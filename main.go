package main

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"go-learn/internal/handler"
	"go-learn/internal/middleware"
	"io/fs"

	authrepo "go-learn/internal/repo/auth"
	orderepo "go-learn/internal/repo/order"
	productrepo "go-learn/internal/repo/product"
	userrepo "go-learn/internal/repo/user"

	authservice "go-learn/internal/service/auth"
	orderservice "go-learn/internal/service/order"
	productservice "go-learn/internal/service/product"
	userservice "go-learn/internal/service/user"

	"log/slog"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq" // Драйвер для Postgres
	"github.com/pressly/goose/v3"
	"github.com/redis/go-redis/v9"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

func main() {

	// === СОЗДАЕМ ЛОГГЕР ===
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	err := godotenv.Load()
	if err != nil {
		logger.Warn("⚠️  .env файл не найден, использую системные переменные")
	}

	// === ЗАГРУЗКА ПЕРЕМЕННЫХ ===
	dbUser := getEnv("DB_USER", "user")
	dbPass := getEnv("DB_PASSWORD", "password")
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbName := getEnv("DB_NAME", "postgres")
	serverPort := getEnv("SERVER_PORT", "8080")
	redisHost := getEnv("REDIS_HOST", "localhost")
	redisPort := getEnv("REDIS_PORT", "6379")
	redisPassword := getEnv("REDIS_PASSWORD", "")
	logger.Info("✅️ Кофигурация приложения выполнена")

	// === ПОДКЛЮЧЕНИЕ К БАЗЕ ДАННЫХ POSTGRES ===
	conStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPass, dbName,
	)

	db, err := sql.Open("postgres", conStr)
	if err != nil {
		logger.Error("Ошибка подключения к базе данных", "error", err)
		os.Exit(1)
	}
	defer db.Close() // Закроем соединение когда программа завершится

	err = db.Ping()
	if err != nil {
		logger.Error("Не удалось подключиться к Postgres", "error", err)
		os.Exit(1)
	}
	logger.Info("✅ Успешно подключено к Postgres!")

	// === ПОДКЛЮЧЕНИЕ К REDIS===
	redisClient := redis.NewClient(&redis.Options{
		Addr:     redisHost + ":" + redisPort,
		Password: redisPassword,
		DB:       0,
	})
	defer func() {
		if err := redisClient.Close(); err != nil {
			logger.Error("Ошибка при закрытии Redis", "error", err)
		}
	}()

	// Проверка соединения с Redis
	ctx := context.Background()
	err = redisClient.Ping(ctx).Err()
	if err != nil {
		logger.Error("Не удалось подключиться к Redis", "error", err)
		os.Exit(1)
	}
	logger.Info("✅ Успешно подключено к Redis!")

	// === НАКАТЫВАЕМ МИГРАЦИЮ
	migrationsFS, err := fs.Sub(embedMigrations, "migrations")
	if err != nil {
		logger.Error("Ошибка извлечения миграций", "error", err)
		os.Exit(1)
	}

	goose.SetBaseFS(migrationsFS)
	if err := goose.Up(db, "."); err != nil {
		logger.Error("Ошибка миграции", "error", err)
		os.Exit(1)
	}
	logger.Info("✅ Миграции успешно применены")

	// === ИНИЦИАЛИЗАЦИЯ СЛОЕВ ===
	orderRepo := orderepo.NewOrderRepository(redisClient, logger)
	productRepo := productrepo.NewProductRepository(db, logger)
	userRepo := userrepo.NewUserRepository(db, logger)
	authRepo := authrepo.NewAuthRepository(db, logger)
	sessionCache := authrepo.NewSessionCache(redisClient)

	orderService := orderservice.NewOrderService(orderRepo, logger)
	productService := productservice.NewProductService(productRepo, logger)
	userService := userservice.NewUserService(userRepo, logger)
	authService := authservice.NewAuthService(authRepo, sessionCache, logger)

	h := handler.NewHandler(productService, orderService, userService, authService, logger)

	// === НАСТРОЙКА РОУТЕРА ===
	router := gin.Default()

	// Публичные эндпоинты
	router.POST("/user/create", h.CreateUser)
	router.POST("/login", h.Login)
	router.GET("/product/:id", h.GetProductById)
	router.GET("/products", h.GetAllProducts)

	// Защищенные эндпоинты (требуют авторизацию)
	auth := router.Group("/")
	auth.Use(middleware.AuthMiddleware(authService))
	{
		auth.POST("/logout", h.Logout)
		auth.DELETE("/product/delete/:id", h.DeleteProductById)
		auth.POST("/product/create", h.CreateProduct)
		auth.POST("/buy/product/:id", h.BuyProduct)
		auth.GET("/product/:id/orders", h.GetCounts)
	}

	// Запуск http-сервиса
	fmt.Println("🌐 Started to http://localhost:" + serverPort)
	err = router.Run(":" + serverPort)
	if err != nil {
		logger.Error("Ошибка создания таблицы", "error", err)
		os.Exit(1)
	}
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
