package ports

import (
	"context"
)

type RedisStream interface {
	PublisherEvent(ctx context.Context, data any) error
}
