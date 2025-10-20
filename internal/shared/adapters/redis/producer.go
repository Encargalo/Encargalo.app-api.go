package redis

import (
	"context"

	"Encargalo.app-api.go/internal/shared/config"
	"github.com/andresxlp/redsumer/pkg/client"
	"github.com/andresxlp/redsumer/pkg/producer"
	"github.com/charmbracelet/log"
)

func NewProducerRedisStreamConnection(config config.Config) *producer.Producer {
	configRedis := config.RedisStream
	redisArgs := client.RedisArgs{
		RedisHost:  configRedis.Host,
		RedisPort:  configRedis.Port,
		Db:         0,
		ClientName: configRedis.User,
		Password:   configRedis.Password,
	}

	producerArgs := producer.ProducerArgs{StreamName: configRedis.StreamName}

	ctx := context.Background()
	newProducer, err := producer.NewProducer(ctx, redisArgs, producerArgs)
	if err != nil {
		log.Fatalf("Error creating new producer %v", err)
	}

	return newProducer
}
