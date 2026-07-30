package main

import (
	"context"
	"database/sql"
	"fmt"
	"go-learn/internal/events/consumers/orders"
	authhandler "go-learn/internal/facade/rest/handler/auth"
	orderhandler "go-learn/internal/facade/rest/handler/order"
	userrepo "go-learn/internal/repo/postgres/sesssion"
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
	serverPort := getEnv("SERVER_PORT", "8080")
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
	if err := goose.Up(db, "."); err != nil {
		logger.Error("Ошибка миграции", "error", err)
		os.Exit(1)
	}
	logger.Info("✅ Миграции успешно применены")

	// === РЕПОЗИТОРИИ ===
	//postgres
	authRepo := userrepo.NewAuthRepository(db, logger)
	// redis
	orderCacheRepo := order.NewOrderCacheRepository(redisClient, logger)
	sessionCache := session.NewSessionCache(redisClient)

	// === KAFKA CONSUMER ===
	brokers := strings.Split(kafkaBrokers, ",")
	productConsumer := orders.NewProductConsumer(brokers, orderCacheRepo, logger)
	defer productConsumer.Close()

	ctx = context.Background()
	productConsumer.Start(ctx)

	// === СЕРВИСЫ ===
	orderCacheService := orderservice.NewOrderCacheService(orderCacheRepo, logger)
	authService := authservice.NewAuthService(authRepo, sessionCache, logger)

	// === ХЕНДЛЕР ===
	h := orderhandler.NewHandler(orderCacheService, logger)

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
