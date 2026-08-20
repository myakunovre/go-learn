package orders

import (
	"context"
	"encoding/json"
	"go-learn/internal/events"
	"go-learn/internal/service/order"
	"log/slog"

	"github.com/segmentio/kafka-go"
)

type ProductConsumer struct {
	reader            *kafka.Reader
	orderService      *order.OrderService
	orderCacheService *order.OrderCacheService
	logger            *slog.Logger
}

func NewProductConsumer(brokers []string, orderService *order.OrderService, orderCacheService *order.OrderCacheService, logger *slog.Logger) *ProductConsumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     brokers,
		Topic:       "core-events",
		GroupID:     "order-service",
		MinBytes:    10e3,
		MaxBytes:    10e6,
		StartOffset: kafka.LastOffset,
	})

	return &ProductConsumer{
		reader:            reader,
		orderService:      orderService,
		orderCacheService: orderCacheService,
		logger:            logger,
	}
}

func (c *ProductConsumer) Start(ctx context.Context) {
	c.logger.Info("Starting Kafka consumer for core events")

	go func() {
		for {
			select {
			case <-ctx.Done():
				c.logger.Info("Stopping Kafka consumer")
				return
			default:
				msg, err := c.reader.ReadMessage(ctx)
				if err != nil {
					if ctx.Err() != nil {
						return
					}

					c.logger.Error("Failed to read message", "error", err)
					continue
				}

				c.handleMessage(ctx, msg)
			}
		}
	}()
}

func (c *ProductConsumer) handleMessage(ctx context.Context, msg kafka.Message) {
	switch string(msg.Key) {
	case "core.deleted":
		var event events.ProductDeleted
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			c.logger.Error("Failed to unmarshal core.deleted", "error", err)
			return
		}
		c.logger.Info("Product deleted event received", "product_id", event.ProductID)

		// Помечаем удаленный товар в DB Order-сервиса
		err := c.orderService.MarkDeletedProduct(ctx, int(event.ProductID))
		if err != nil {
			c.logger.Error("Failed to mark deleted core", "error", err)
		}

		// Очищаем кэш заказов для удаленного продукта
		err = c.orderCacheService.DeleteProduct(ctx, event.ProductID)
		if err != nil {
			c.logger.Error("Failed to delete core from OrderCache", "error", err)
		}

	default:
		c.logger.Warn("Unknown event type", "key", string(msg.Key))
	}
}

func (c *ProductConsumer) Close() error {
	return c.reader.Close()
}
