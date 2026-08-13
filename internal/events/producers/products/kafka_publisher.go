package products

import (
	"context"
	"encoding/json"
	"go-learn/internal/events"
	"log/slog"

	"github.com/segmentio/kafka-go"
)

type KafkaEventPublisher struct {
	writer *kafka.Writer
	logger *slog.Logger
}

func NewKafkaEventPublisher(brokers []string, logger *slog.Logger) *KafkaEventPublisher {
	writer := &kafka.Writer{
		Addr:     kafka.TCP(brokers...),
		Balancer: &kafka.LeastBytes{},
	}

	return &KafkaEventPublisher{
		writer: writer,
		logger: logger,
	}
}

func (p *KafkaEventPublisher) Publish(ctx context.Context, topic string, event events.Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		p.logger.Error("Failed to marshal event", "error", err)
		return err
	}

	msg := kafka.Message{
		Topic: topic,
		Key:   []byte(event.GetType()),
		Value: data,
	}

	err = p.writer.WriteMessages(ctx, msg)
	if err != nil {
		p.logger.Error("Failed to publish to Kafka", "topic", topic, "error", err)
		return err
	}

	p.logger.Info("Event published to Kafka", "topic", topic, "event_type", event.GetType())
	return nil
}

func (p *KafkaEventPublisher) Close() error {
	return p.writer.Close()
}
