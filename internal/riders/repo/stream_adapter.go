package repo

import (
	"context"

	ordersPorts "Encargalo.app-api.go/internal/orders/domain/ports"
	ridersPorts "Encargalo.app-api.go/internal/riders/domain/ports"
)

type ridersStreamAdapter struct {
	stream ordersPorts.RedisStream
}

func NewRidersStream(stream ordersPorts.RedisStream) ridersPorts.RidersStream {
	return &ridersStreamAdapter{stream}
}

func (r *ridersStreamAdapter) PublisherEvent(ctx context.Context, data any) error {
	return r.stream.PublisherEvent(ctx, data)
}
