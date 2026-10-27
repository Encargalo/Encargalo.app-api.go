package redis

import (
	"context"
	"log/slog"
	"net/url"
	"os"
	"strconv"
	"time"

	"Encargalo.app-api.go/internal/shared/config"
	"github.com/andresxlp/redsumer/pkg/client"
	"github.com/andresxlp/redsumer/pkg/consumer"
)

// NewConsumerRedisStreamConnection conecta un consumer al stream compartido
// usando el group/consumer propios del worker que llama (cmd/notifications,
// cmd/etl, ...) — cada uno lee el mismo stream de forma independiente.
func NewConsumerRedisStreamConnection(ctx context.Context, redisURL string, configRedis config.RedisConsumerGroup) *consumer.Consumer {
	u, err := url.Parse(redisURL)
	if err != nil {
		slog.Error("error parseando redis URL", "error", err)
		os.Exit(1)
	}
	port, _ := strconv.Atoi(u.Port())
	password, _ := u.User.Password()

	redisArgs := client.RedisArgs{
		RedisHost:  u.Hostname(),
		RedisPort:  port,
		Db:         configRedis.Db,
		ClientName: u.User.Username(),
		Password:   password,
	}

	var claimBatchSize int64 = 1
	var pendingBatchSize int64 = 1
	consumerArgs := consumer.ConsumerArgs{
		StreamName:         configRedis.StreamName,
		GroupName:          configRedis.GroupName,
		ConsumerName:       configRedis.ConsumerName,
		BatchSize:          10,
		ClaimBatchSize:     &claimBatchSize,
		PendingBatchSize:   &pendingBatchSize,
		Block:              time.Second * 1,
		MinDurationToClaim: time.Second * 1,
		IdleStillMine:      0,
		Tries:              []int{1, 2, 3, 10, 15},
	}

	consumerClient := connectWithRetry(ctx, redisArgs, consumerArgs)

	slog.Info("consumer conectado",
		"stream_name", configRedis.StreamName,
		"group_name", configRedis.GroupName,
		"consumer_name", configRedis.ConsumerName,
	)

	return consumerClient
}

// connectWithRetry reintenta con backoff en vez de abortar el proceso: Redis en
// Railway puede estar "dormido" (sleep por inactividad) y el primer intento de
// conexión falla mientras despierta — un panic() ahí hace que el contenedor
// reinicie una y otra vez contra el mismo fallo (crash-loop, inunda los logs).
func connectWithRetry(ctx context.Context, redisArgs client.RedisArgs, consumerArgs consumer.ConsumerArgs) *consumer.Consumer {
	const maxBackoff = 30 * time.Second
	backoff := time.Second

	for {
		consumerClient, err := consumer.NewConsumer(ctx, redisArgs, consumerArgs)
		if err == nil {
			return consumerClient
		}

		slog.Error("error conectando a redis, reintentando", "error", err, "retry_in", backoff.String())
		time.Sleep(backoff)
		if backoff < maxBackoff {
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}
	}
}
