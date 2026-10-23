package repo

import (
	"context"

	financeModels "Encargalo.app-api.go/internal/finance/domain/models"
	ordersPorts "Encargalo.app-api.go/internal/orders/domain/ports"
	"Encargalo.app-api.go/internal/riders/domain/models"
	"Encargalo.app-api.go/internal/riders/domain/ports"
)

type withdrawalStream struct {
	stream ordersPorts.RedisStream
}

// NewWithdrawalPublisher reusa el productor del stream que ya comparte el
// monorepo: el consumer de notificaciones lee de ahí.
func NewWithdrawalPublisher(stream ordersPorts.RedisStream) ports.WithdrawalPublisher {
	return &withdrawalStream{stream: stream}
}

func (w *withdrawalStream) PublishWithdrawalRequested(ctx context.Context, req financeModels.WithdrawalRequest) error {
	event := models.DataEventWithdrawalRequested{}
	event.BuildFrom(req)
	return w.stream.PublisherEvent(ctx, event)
}
