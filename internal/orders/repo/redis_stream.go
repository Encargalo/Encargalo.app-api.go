package repo

import (
	"context"

	"Encargalo.app-api.go/internal/orders/domain/models"
	"Encargalo.app-api.go/internal/orders/domain/ports"
	"Encargalo.app-api.go/internal/pkg/json"
	"github.com/andresxlp/redsumer/pkg/producer"
)

type redisRepository struct {
	producer *producer.Producer
}

func NewRepositoryProducerStream(producer *producer.Producer) ports.RedisStream {
	return &redisRepository{producer}
}

func (stream *redisRepository) Producer(ctx context.Context, items []models.DataProduceOrder) error {

	for _, v := range items {
		message, err := json.StructToMap(v)
		if err != nil {
			return err
		}

		if err = stream.producer.Produce(ctx, message); err != nil {
			return err
		}
	}

	return nil
}
