package cache

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

type Cache interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, val interface{}, time time.Duration) error
}

type cache struct {
	redis *redis.Client
}

func NewCache(redis *redis.Client) Cache {
	return &cache{
		redis: redis,
	}
}

func (c *cache) Get(ctx context.Context, key string) (string, error) {
	return c.redis.Get(ctx, key).Result()
}

func (c *cache) Set(ctx context.Context, key string, val interface{}, time time.Duration) error {

	result, err := json.Marshal(val)
	if err != nil {
		slog.Error("error al hacer marshal de los products", "error", err)
	}

	return c.redis.Set(ctx, key, result, time).Err()
}
