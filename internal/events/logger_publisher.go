package events

import (
	"context"
	"log/slog"
)

type LoggerEventPublisher struct {
	logger *slog.Logger
}

func NewLoggerEventPublisher(logger *slog.Logger) *LoggerEventPublisher {
	return &LoggerEventPublisher{logger: logger}
}

func (p *LoggerEventPublisher) Publish(ctx context.Context, topic string, event Event) error {
	p.logger.Info("Publishing event",
		"topic", topic,
		"event_type", event.GetType(),
		"event", event,
	)
	// Здесь можно вернуть nil, т.к. это заглушка
	return nil
}
