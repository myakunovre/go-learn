package orders

import (
	"context"
	"encoding/json"
	"go-learn/internal/events"
	"log/slog"

	"github.com/segmentio/kafka-go"
)

type ProductConsumer struct {
	reader *kafka.Reader
	logger *slog.Logger
}

func NewProductConsumer(brokers []string, logger *slog.Logger) *ProductConsumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     brokers,
		Topic:       "product-events",
		GroupID:     "order-service",
		MinBytes:    10e3,
		MaxBytes:    10e6,
		StartOffset: kafka.LastOffset,
	})

	return &ProductConsumer{
		reader: reader,
		logger: logger,
	}
}

func (c *ProductConsumer) Start(ctx context.Context) {
	c.logger.Info("Starting Kafka consumer for product events")

	go func() {
		for {
			select {
			case <-ctx.Done():
				c.logger.Info("Stopping Kafka consumer")
				return
			default:
				msg, err := c.reader.ReadMessage(ctx)
				if err != nil {
					c.logger.Error("Failed to read message", "error", err)
					continue
				}

				c.handleMessage(msg)
			}
		}
	}()
}

func (c *ProductConsumer) handleMessage(msg kafka.Message) {
	switch string(msg.Key) {
	case "product.deleted":
		var event events.ProductDeleted
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			c.logger.Error("Failed to unmarshal product.deleted", "error", err)
			return
		}
		c.logger.Info("Product deleted event received", "product_id", event.ProductID)
		// Здесь можно очистить кэш заказов для удаленного продукта

	default:
		c.logger.Warn("Unknown event type", "key", string(msg.Key))
	}
}

func (c *ProductConsumer) Close() error {
	return c.reader.Close()
}
