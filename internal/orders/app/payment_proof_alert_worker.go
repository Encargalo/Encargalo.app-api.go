package app

import (
	"context"
	"log/slog"
	"time"

	"Encargalo.app-api.go/internal/orders/domain/ports"
)

const paymentProofAlertWorkerInterval = 30 * time.Second

// PaymentProofAlertWorker publica payment_proof_pending al crear (lo hace
// CreateForOrder) y cada 3 minutos mientras el comprobante siga pendiente.
type PaymentProofAlertWorker struct {
	reviews ports.PaymentReviewsApp
}

func NewPaymentProofAlertWorker(reviews ports.PaymentReviewsApp) *PaymentProofAlertWorker {
	return &PaymentProofAlertWorker{reviews: reviews}
}

func (w *PaymentProofAlertWorker) Start(ctx context.Context) {
	go w.loop(ctx)
}

func (w *PaymentProofAlertWorker) loop(ctx context.Context) {
	ticker := time.NewTicker(paymentProofAlertWorkerInterval)
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

func (w *PaymentProofAlertWorker) tick(ctx context.Context) {
	if err := w.reviews.NotifyPendingPaymentProofs(ctx); err != nil {
		slog.Error("PaymentProofAlertWorker tick failed", "error", err)
	}
}
