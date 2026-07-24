package redis

import (
	"context"
	"net/url"
	"strconv"

	"Encargalo.app-api.go/internal/shared/config"
	"github.com/andresxlp/redsumer/pkg/client"
	"github.com/andresxlp/redsumer/pkg/producer"
	"github.com/charmbracelet/log"
)

func NewProducerRedisStreamConnection(config config.Config) *producer.Producer {
	u, err := url.Parse(config.Redis.URL)
	if err != nil {
		log.Fatalf("Error parsing redis URL: %v", err)
	}
	port, _ := strconv.Atoi(u.Port())
	password, _ := u.User.Password()

	redisArgs := client.RedisArgs{
		RedisHost:  u.Hostname(),
		RedisPort:  port,
		Db:         config.RedisStream.Db,
		ClientName: u.User.Username(),
		Password:   password,
	}

	producerArgs := producer.ProducerArgs{StreamName: config.RedisStream.StreamName}

	ctx := context.Background()
	newProducer, err := producer.NewProducer(ctx, redisArgs, producerArgs)
	if err != nil {
		log.Fatalf("Error creating new producer %v", err)
	}

	return newProducer
}
