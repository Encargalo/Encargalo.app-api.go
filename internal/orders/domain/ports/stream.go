package ports

import (
	"context"

	"Encargalo.app-api.go/internal/orders/domain/models"
)

type RedisStream interface {
	Producer(ctx context.Context, items []models.DataProduceOrder) error
}
