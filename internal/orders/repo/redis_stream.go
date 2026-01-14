package repo

import (
	"context"
	"reflect"

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

func (stream *redisRepository) PublisherEvent(ctx context.Context, data interface{}) error {

	if stream.isSlice(data) {

		for _, v := range data.([]interface{}) {
			message, err := json.StructToMap(v)
			if err != nil {
				return err
			}

			if err = stream.producer.Produce(ctx, message); err != nil {
				return err
			}
		}
	}

	message, err := json.StructToMap(data)
	if err != nil {
		return err
	}

	if err = stream.producer.Produce(ctx, message); err != nil {
		return err
	}

	return nil
}

func (stream *redisRepository) isSlice(i interface{}) bool {
	v := reflect.ValueOf(i)
	return v.Kind() == reflect.Slice
}
