package metrics

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// Общее количество запросов
	RequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)
	// Длительность запросов (гистограмма)
	RequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds",
			Buckets: prometheus.DefBuckets, // можно настроить
		},
		[]string{"method", "path"},
	)
)

func init() {
	// Регистрируем метрики
	prometheus.MustRegister(RequestsTotal, RequestDuration)
}

// PrometheusMiddleware возвращает gin.HandlerFunc для сбора метрик
func PrometheusMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		// Обработка запроса
		c.Next()
		// После ответа
		duration := time.Since(start).Seconds()
		status := strconv.Itoa(c.Writer.Status())
		path := c.FullPath() // например, "/product/:id"
		method := c.Request.Method

		// Увеличиваем счётчик запросов
		RequestsTotal.WithLabelValues(method, path, status).Inc()
		// Записываем длительность
		RequestDuration.WithLabelValues(method, path).Observe(duration)
	}
}
