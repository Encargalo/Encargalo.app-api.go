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

func (stream *redisRepository) PublisherEvent(ctx context.Context, data any) error {
	items, err := normalizeToSlice(data)
	if err != nil {
		return err
	}

	for _, v := range items {
		message, err := json.StructToMap(v)
		if err != nil {
			return err
		}

		if err := stream.producer.Produce(ctx, message); err != nil {
			return err
		}
	}

	return nil
}

func normalizeToSlice(data any) ([]any, error) {
	v := reflect.ValueOf(data)

	if v.Kind() != reflect.Slice {
		return []any{data}, nil
	}

	out := make([]any, v.Len())
	for i := 0; i < v.Len(); i++ {
		out[i] = v.Index(i).Interface()
	}

	return out, nil
}
