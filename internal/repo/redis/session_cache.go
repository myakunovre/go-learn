package redis

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type SessionCache struct {
	client *redis.Client
}

func NewSessionCache(client *redis.Client) *SessionCache {
	return &SessionCache{client: client}
}

func (c *SessionCache) SetSession(ctx context.Context, token string, userID int, ttl time.Duration) error {
	key := fmt.Sprintf("session:%s", token)
	return c.client.Set(ctx, key, userID, ttl).Err()
}

func (c *SessionCache) GetSession(ctx context.Context, token string) (int, error) {
	key := fmt.Sprintf("session:%s", token)
	val, err := c.client.Get(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(val)
}

func (c *SessionCache) DeleteSession(ctx context.Context, token string) error {
	key := fmt.Sprintf("session:%s", token)
	return c.client.Del(ctx, key).Err()
}
