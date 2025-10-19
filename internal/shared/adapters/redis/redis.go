package redis

import (
	"Encargalo.app-api.go/internal/shared/config"
	"github.com/redis/go-redis/v9"
)

var RedisClient *redis.Client

func NewRedisConnection(config config.Config) *redis.Client {
	configRedis := config.Redis
	redisArg, err := redis.ParseURL(configRedis.URL)
	if err != nil {
		panic(err)
	}

	RedisClient = redis.NewClient(redisArg)

	return RedisClient
}
