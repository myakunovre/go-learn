package main

import (
	"context"
	"database/sql"
	"fmt"
	"go-learn/internal/events/consumers/orders"
	"go-learn/internal/facade/grpc/core"
	auth2 "go-learn/internal/facade/rest/handler/auth"
	orderhandler "go-learn/internal/facade/rest/handler/order"
	orderrepo "go-learn/internal/repo/postgres/order"
	"go-learn/internal/repo/redis/order"
	"go-learn/internal/repo/redis/session"
	authservice "go-learn/internal/service/core/auth"
	"go-learn/migrations"
	"strings"

	orderservice "go-learn/internal/service/order"

	"net/http"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"log/slog"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq" // Драйвер для Postgres
	"github.com/pressly/goose/v3"
	redislib "github.com/redis/go-redis/v9"

	"go-learn/internal/metrics"

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
	serverPort := getEnv("ORDER_SERVER_PORT", "8082")
	//Postgres
	dbUser := getEnv("ORDER_DB_USER", "user")
	dbPass := getEnv("ORDER_DB_ORDER_PASSWORD", "password")
	dbHost := getEnv("ORDER_DB_HOST", "localhost")
	dbPort := getEnv("ORDER_DB_PORT", "5433")
	dbName := getEnv("ORDER_DB_NAME", "postgres")
	//Redis
	redisHost := getEnv("REDIS_HOST", "localhost")
	redisPort := getEnv("REDIS_PORT", "6379")
	redisDB := getEnvInt("CORE_REDIS_DB", 1)
	redisPassword := getEnv("REDIS_PASSWORD", "")
	//Kafka
	kafkaBrokers := getEnv("KAFKA_BROKERS", "localhost:9092")
	//GRPC
	grpcPort := getEnv("CORE_GRPC_PORT", "50051")
	//Graceful Shutdown
	shutdownTimeout := getEnvInt("SHUTDOWN_TIMEOUT", 5)

	logger.Info("✅️ [Orders] Конфигурация приложения выполнена")

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
		DB:       redisDB,
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
	goose.SetBaseFS(migrations.OrderMigrationsFS)
	if err := goose.Up(db, "order"); err != nil {
		logger.Error("Ошибка миграции", "error", err)
		os.Exit(1)
	}
	logger.Info("✅ Миграции успешно применены")

	// === РЕПОЗИТОРИИ ===
	//postgres
	orderRepo := orderrepo.NewOrderRepository(db, logger)
	// todo: для authRepo требуются userRepo. Но эта БД в core-сервисе.
	// todo: как быть? Я же не могу в order-сервсие использовать БД core-сервиса
	authRepo := userrepo.NewAuthRepository(db, logger)
	// redis
	orderCacheRepo := order.NewOrderCacheRepository(redisClient, logger)
	sessionCache := session.NewSessionCache(redisClient)

	// Запуск GRPC-клиента
	productGRPCClient, err := core.NewProductGRPCClient(grpcPort, logger)
	if err != nil {
		logger.Error("Failed to create gRPC client", "error", err)
		os.Exit(1)
	}
	defer productGRPCClient.Close()

	// todo: добавить gRPC сервер для запросов от core (сначала придумать зачем core-сервису забронированные товары)

	// === СЕРВИСЫ ===
	orderService := orderservice.NewOrderService(orderRepo, productGRPCClient, logger)
	orderCacheService := orderservice.NewOrderCacheService(orderCacheRepo, logger)
	authService := authservice.NewAuthService(authRepo, sessionCache, logger)

	// === KAFKA CONSUMER ===
	brokers := strings.Split(kafkaBrokers, ",")
	productConsumer := orders.NewProductConsumer(brokers, orderService, orderCacheService, logger)
	defer productConsumer.Close()

	ctx = context.Background()
	productConsumer.Start(ctx)

	// === ХЕНДЛЕР ===
	cacheHandler := orderhandler.NewCacheHandler(orderCacheService, logger)
	orderHandler := orderhandler.NewHandler(orderService, logger)
	authHandler := auth2.NewHandler(authService, logger)

	// === НАСТРОЙКА РОУТЕРА ===
	router := gin.Default()

	// Подключаем middleware для сбора метрик (перед всеми маршрутами)
	router.Use(metrics.PrometheusMiddleware())

	// Эндпоинт для Prometheus
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Публичные эндпоинты order cache
	// todo: наверное надо POST убрать из REST API, т.к. вряд ли такой эндпоинт нужен клиенту?
	// todo: как будто логично дергать метод BuyProduct в сервисном слое (из какого-то другого сервиса)
	router.POST("/products/:id/buy", cacheHandler.BuyProduct)
	router.GET("/products/:id/orders", cacheHandler.GetCounts)

	// Публичные эндпоинты order DB
	//todo: Проверить эндпоинты order-сервиса (нужна ли авторизация в этом сервисе?)
	router.GET("/orders/:id", orderHandler.GetOrderById)
	router.POST("/login", authHandler.Login)

	// Защищенные эндпоинты (требуют авторизацию)
	auth := router.Group("/")
	auth.Use(auth2.AuthMiddleware(authService))
	{
		auth.POST("/logout", authHandler.Logout)
		auth.POST("/orders/create", orderHandler.CreateOrder)
		auth.DELETE("/orders/:id", orderHandler.DeleteOrderById)
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
