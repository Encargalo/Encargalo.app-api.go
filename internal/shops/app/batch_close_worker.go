package app

import (
	"context"
	"log/slog"
	"time"

	portsShops "Encargalo.app-api.go/internal/shops/domain/ports"
)

const batchCloseWorkerInterval = 45 * time.Second

// BatchCloseWorker cierra lotes de locales cerrados sin órdenes activas.
type BatchCloseWorker struct {
	repo portsShops.ShopsRepo
}

func NewBatchCloseWorker(repo portsShops.ShopsRepo) *BatchCloseWorker {
	return &BatchCloseWorker{repo: repo}
}

func (w *BatchCloseWorker) Start(ctx context.Context) {
	go w.loop(ctx)
}

func (w *BatchCloseWorker) loop(ctx context.Context) {
	ticker := time.NewTicker(batchCloseWorkerInterval)
	defer ticker.Stop()

	w.tick(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.tick(ctx)
		}
	}
}

func (w *BatchCloseWorker) tick(ctx context.Context) {
	if err := w.repo.TryCloseAllEligibleBatches(ctx); err != nil {
		slog.Error("BatchCloseWorker tick failed", "error", err)
	}
}
