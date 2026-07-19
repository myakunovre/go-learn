package main

import (
	"context"
	"database/sql"
	"fmt"
	"go-learn/internal/events"
	authhandler "go-learn/internal/facade/rest/handler/auth"
	orderhandler "go-learn/internal/facade/rest/handler/order"
	userrepo "go-learn/internal/repo/postgres/sesssion"
	"go-learn/internal/repo/redis/order"
	"go-learn/internal/repo/redis/session"
	"strings"

	authservice "go-learn/internal/service/auth"
	orderservice "go-learn/internal/service/order"

	"go-learn/migrations"
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
	dbUser := getEnv("DB_ORDER_USER", "user")
	dbPass := getEnv("DB_ORDER_PASSWORD", "password")
	dbHost := getEnv("DB_ORDER_HOST", "localhost")
	dbPort := getEnv("DB_ORDER_PORT", "5433")
	dbName := getEnv("DB_ORDER_NAME", "postgres")
	serverPort := getEnv("SERVER_PORT", "8080")
	redisHost := getEnv("REDIS_HOST", "localhost")
	redisPort := getEnv("REDIS_PORT", "6379")
	redisPassword := getEnv("REDIS_PASSWORD", "")
	kafkaBrokers := getEnv("KAFKA_BROKERS", "localhost:9092")
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
	goose.SetBaseFS(migrations.MigrationsFS)
	if err := goose.Up(db, "."); err != nil {
		logger.Error("Ошибка миграции", "error", err)
		os.Exit(1)
	}
	logger.Info("✅ Миграции успешно применены")

	// === РЕПОЗИТОРИИ ===
	orderRepo := order.NewOrderRepository(redisClient, logger)
	authRepo := userrepo.NewAuthRepository(db, logger)
	sessionCache := session.NewSessionCache(redisClient)

	// === EVENT PUBLISHER (KAFKA) ===
	brokers := strings.Split(kafkaBrokers, ",")
	eventPublisher := events.NewKafkaEventPublisher(brokers, logger)
	defer func() {
		if err := eventPublisher.Close(); err != nil {
			logger.Error("Ошибка при закрытии Kafka publisher", "error", err)
		}
	}()
	logger.Info("✅ Kafka publisher создан", "brokers", brokers)

	// === KAFKA CONSUMER ===
	productConsumer := events.NewProductConsumer(brokers, logger)
	defer productConsumer.Close()

	ctx = context.Background()
	productConsumer.Start(ctx)

	// === СЕРВИСЫ ===
	orderService := orderservice.NewOrderService(orderRepo, logger)
	authService := authservice.NewAuthService(authRepo, sessionCache, logger)

	// === ХЕНДЛЕР ===
	h := orderhandler.NewHandler(orderService, logger)

	// === НАСТРОЙКА РОУТЕРА ===
	router := gin.Default()

	// Подключаем middleware для сбора метрик (перед всеми маршрутами)
	router.Use(metrics.PrometheusMiddleware())

	// Эндпоинт для Prometheus
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Защищенные эндпоинты (требуют авторизацию)
	auth := router.Group("/")
	auth.Use(authhandler.AuthMiddleware(authService))
	{
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
