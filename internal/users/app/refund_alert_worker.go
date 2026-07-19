package app

import (
	"context"
	"log/slog"
	"time"

	"Encargalo.app-api.go/internal/users/domain/ports"
)

const refundAlertWorkerInterval = 30 * time.Second

// RefundAlertWorker publica balance_refund_pending cada 5 minutos
// mientras la solicitud siga pending o accepted.
type RefundAlertWorker struct {
	finance ports.CustomerFinanceApp
}

func NewRefundAlertWorker(finance ports.CustomerFinanceApp) *RefundAlertWorker {
	return &RefundAlertWorker{finance: finance}
}

func (w *RefundAlertWorker) Start(ctx context.Context) {
	go w.loop(ctx)
}

func (w *RefundAlertWorker) loop(ctx context.Context) {
	ticker := time.NewTicker(refundAlertWorkerInterval)
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

func (w *RefundAlertWorker) tick(ctx context.Context) {
	if err := w.finance.NotifyIncompleteRefunds(ctx); err != nil {
		slog.Error("RefundAlertWorker tick failed", "error", err)
	}
}
