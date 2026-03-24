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
	Set(ctx context.Context, key string, val any, time time.Duration) error
	Clear(ctx context.Context, key string) error
	ClearByPrefix(ctx context.Context, prefix string) error
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

func (c *cache) Set(ctx context.Context, key string, val any, time time.Duration) error {

	result, err := json.Marshal(val)
	if err != nil {
		slog.Error("error al hacer marshal de los products", "error", err)
	}

	return c.redis.Set(ctx, key, result, time).Err()
}

func (c *cache) Clear(ctx context.Context, key string) error {
	return c.redis.Del(ctx, key).Err()
}

func (c *cache) ClearByPrefix(ctx context.Context, prefix string) error {
	keys, err := c.redis.Keys(ctx, prefix+"*").Result()
	if err != nil || len(keys) == 0 {
		return err
	}
	return c.redis.Del(ctx, keys...).Err()
}
