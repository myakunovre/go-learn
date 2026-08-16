package main

import (
	"context"
	"database/sql"
	"fmt"
	"go-learn/internal/events/producers/products"
	"go-learn/internal/facade/grpc/core"
	auth2 "go-learn/internal/facade/rest/handler/auth"
	producthandler "go-learn/internal/facade/rest/handler/product"
	userhandler "go-learn/internal/facade/rest/handler/user"
	"go-learn/internal/repo/postgres/product"
	"go-learn/internal/repo/postgres/sesssion"
	userrepo "go-learn/internal/repo/postgres/user"
	"go-learn/internal/repo/redis/session"
	authservice "go-learn/internal/service/core/auth"
	productservice "go-learn/internal/service/core/product"
	userservice "go-learn/internal/service/core/user"
	"go-learn/migrations"
	"net/http"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"log/slog"
	"os"

	"go-learn/internal/metrics"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq" // Драйвер для Postgres
	"github.com/pressly/goose/v3"
	redislib "github.com/redis/go-redis/v9"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {

	// === СОЗДАЕМ ЛОГГЕР ===
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	// === ЗАГРУЖАЕМ НАСТРОЙКИ ИЗ ENV-ФАЙЛА ===
	if err := godotenv.Load(); err != nil {
		logger.Warn("⚠️  .env файл не найден, использую системные переменные")
	}

	// === ЗАГРУЗКА ПЕРЕМЕННЫХ ===
	serverPort := getEnv("CORE_SERVER_PORT", "8080")
	//Postgres
	dbUser := getEnv("CORE_DB_USER", "user")
	dbPass := getEnv("CORE_DB_PASSWORD", "password")
	dbHost := getEnv("CORE_DB_HOST", "localhost")
	dbPort := getEnv("CORE_DB_PORT", "5432")
	dbName := getEnv("CORE_DB_NAME", "postgres")
	//Redis
	redisHost := getEnv("CORE_REDIS_HOST", "localhost")
	redisPort := getEnv("CORE_REDIS_PORT", "6379")
	redisDB := getEnvInt("CORE_REDIS_DB", 0)
	redisPassword := getEnv("CORE_REDIS_PASSWORD", "")
	//Kafka
	kafkaBrokers := getEnv("KAFKA_BROKERS", "localhost:9092")
	//GRPC
	grpcPort := getEnv("CORE_GRPC_PORT", "50051")
	//Graceful Shutdown
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

	dbCtx, dbCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer dbCancel()

	err = db.PingContext(dbCtx)
	if err != nil {
		logger.Error("Не удалось подключиться к Postgres", "error", err)
		os.Exit(1)
	}
	logger.Info("✅ Успешно подключено к Postgres!")

	// === ПОДКЛЮЧЕНИЕ К REDIS===
	redisClient := redislib.NewClient(&redislib.Options{
		Addr:     redisHost + ":" + redisPort,
		Password: redisPassword,
		DB:       redisDB,
	})
	defer func() {
		if err := redisClient.Close(); err != nil {
			logger.Error("Ошибка при закрытии Redis", "error", err)
		}
	}()

	// Проверка соединения с Redis
	redisCtx, redisCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer redisCancel()

	err = redisClient.Ping(redisCtx).Err()
	if err != nil {
		logger.Error("Не удалось подключиться к Redis", "error", err)
		os.Exit(1)
	}
	logger.Info("✅ Успешно подключено к Redis!")

	// === НАКАТЫВАЕМ МИГРАЦИЮ
	goose.SetBaseFS(migrations.CoreMigrationsFS)
	if err := goose.Up(db, "core"); err != nil {
		logger.Error("Ошибка миграции", "error", err)
		os.Exit(1)
	}
	logger.Info("✅ Миграции успешно применены")

	// === РЕПОЗИТОРИИ ===
	//postgres
	productRepo := product.NewProductRepository(db, logger)
	userRepo := userrepo.NewUserRepository(db, logger)
	authRepo := sesssion.NewAuthRepository(db, logger)
	// redis
	sessionCache := session.NewSessionCache(redisClient)

	// === EVENT PUBLISHER (KAFKA) ===
	brokers := strings.Split(kafkaBrokers, ",")
	productEventPublisher := products.NewKafkaEventPublisher(brokers, logger)
	defer func() {
		if err := productEventPublisher.Close(); err != nil {
			logger.Error("Ошибка при закрытии Kafka publisher", "error", err)
		}
	}()
	logger.Info("✅ Kafka publisher создан", "brokers", brokers)

	// === СЕРВИСЫ ===
	productService := productservice.NewProductService(productRepo, logger, productEventPublisher)
	userService := userservice.NewUserService(userRepo, logger)
	authService := authservice.NewAuthService(authRepo, sessionCache, logger)

	// Запуск GRPC-сервера
	productGRPCServer := core.NewProductGRPCServer(productService, logger)

	// todo: добавить gRPC клиент для запросов в order (когда будет БТ по запросу забронированных товаров из order-сервиса)

	go func() {
		if err := productGRPCServer.Start(grpcPort); err != nil {
			logger.Error("gRPC server error", "error", err)
			os.Exit(1)
		}
	}()
	logger.Info("✅ gRPC server started", "port", grpcPort)

	// === ХЕНДЛЕР ===
	ph := producthandler.NewHandler(productService, logger)
	uh := userhandler.NewHandler(userService, logger)
	ah := auth2.NewHandler(authService, logger)

	// === НАСТРОЙКА РОУТЕРА ===
	router := gin.Default()

	// Подключаем middleware для сбора метрик (перед всеми маршрутами)
	router.Use(metrics.PrometheusMiddleware())

	// Эндпоинт для Prometheus
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Публичные эндпоинты
	router.POST("/user/create", uh.CreateUser)
	router.POST("/login", ah.Login)
	router.GET("/product/:id", ph.GetProductById)
	router.GET("/products", ph.GetAllProducts)

	// Защищенные эндпоинты (требуют авторизацию)
	auth := router.Group("/")
	auth.Use(auth2.AuthMiddleware(authService))
	{
		auth.POST("/logout", ah.Logout)
		auth.DELETE("/product/delete/:id", ph.DeleteProductById)
		auth.POST("/product/create", ph.CreateProduct)
		auth.PUT("/product/add", ph.AddProduct)
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

	// === Завершение HTTP сервера ===
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
