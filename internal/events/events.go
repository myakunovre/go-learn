package events

import (
	"context"
)

// Event — базовый интерфейс для всех событий
type Event interface {
	GetType() string
}

// EventPublisher — интерфейс публикации событий
type EventPublisher interface {
	Publish(ctx context.Context, topic string, event Event) error
}

// EventConsumer — интерфейс потребления событий
type EventConsumer interface {
	Subscribe(ctx context.Context, topic string, event Event) error
}
