package main

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"go-learn/internal/events"
	"go-learn/internal/handler"
	userrepo "go-learn/internal/repo/postgres"
	"go-learn/internal/repo/redis"
	"io/fs"
	"net/http"
	"os/signal"
	"strconv"
	"syscall"
	"time"

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
	redislib "github.com/redis/go-redis/v9"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go-learn/internal/metrics"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

func main() {

	// === СОЗДАЕМ ЛОГГЕР ===
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	if err := godotenv.Load(); err != nil {
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
	shutdownTimeout := getEnvInt("SHUTDOWN_TIMEOUT", 5)

	logger.Info("✅️ Конфигурация приложения выполнена")

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
	defer db.Close()

	err = db.Ping()
	if err != nil {
		logger.Error("Не удалось подключиться к Postgres", "error", err)
		os.Exit(1)
	}
	logger.Info("✅ Успешно подключено к Postgres!")

	// === ПОДКЛЮЧЕНИЕ К REDIS===
	redisClient := redislib.NewClient(&redislib.Options{
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

	// === РЕПОЗИТОРИИ ===
	orderRepo := redis.NewOrderRepository(redisClient, logger)
	productRepo := userrepo.NewProductRepository(db, logger)
	userRepo := userrepo.NewUserRepository(db, logger)
	authRepo := userrepo.NewAuthRepository(db, logger)
	sessionCache := redis.NewSessionCache(redisClient)

	// === СЕРВИСЫ ===
	orderService := orderservice.NewOrderService(orderRepo, logger)
	eventPublisher := events.NewLoggerEventPublisher(logger)
	productService := productservice.NewProductService(productRepo, logger, eventPublisher)
	userService := userservice.NewUserService(userRepo, logger)
	authService := authservice.NewAuthService(authRepo, sessionCache, logger)

	// === ХЕНДЛЕР ===
	h := handler.NewHandler(productService, orderService, userService, authService, logger)

	// === НАСТРОЙКА РОУТЕРА ===
	router := gin.Default()

	// Подключаем middleware для сбора метрик (перед всеми маршрутами)
	router.Use(metrics.PrometheusMiddleware())

	// Эндпоинт для Prometheus
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Публичные эндпоинты
	router.POST("/user/create", h.CreateUser)
	router.POST("/login", h.Login)
	router.GET("/product/:id", h.GetProductById)
	router.GET("/products", h.GetAllProducts)

	// Защищенные эндпоинты (требуют авторизацию)
	auth := router.Group("/")
	auth.Use(handler.AuthMiddleware(authService))
	{
		auth.POST("/logout", h.Logout)
		auth.DELETE("/product/delete/:id", h.DeleteProductById)
		auth.POST("/product/create", h.CreateProduct)
		auth.POST("/buy/product/:id", h.BuyProduct)
		auth.GET("/product/:id/orders", h.GetCounts)
	}

	// === GRACEFUL SHUTDOWN ===

	// Создаем http-сервера с настройками
	srv := &http.Server{
		Addr:         ":" + serverPort,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Запуск http-сервиса в горутине
	go func() {
		logger.Info("🌐 Started to http://localhost:" + serverPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Ошибка запуска сервера", "error", err)
			os.Exit(1)
		}
	}()

	// === ОЖИДАНИЕ СИГНАЛА ===
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("🛑 Получен сигнал завершения, начинаем graceful shutdown...")
	logger.Info(fmt.Sprintf("⏳ Ожидание завершения запросов (макс. %d сек)", shutdownTimeout))

	// Создаем контекст с тайм-аутом
	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Duration(shutdownTimeout)*time.Second)
	defer cancel()

	// === Завершение ===
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("❌ Принудительное завершение сервера", "error", err)
	} else {
		logger.Info("✅ Все запросы успешно завершены")
	}
	logger.Info("✅ Сервер остановлен")
}

// === ВСПОМОГАТЕЛЬНЫЕ ФУНКЦИИ ===

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getEnvInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	result, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return result
}
