package app

import (
	"context"
	"log/slog"
	"time"

	"Encargalo.app-api.go/internal/orders/domain/models"
	"Encargalo.app-api.go/internal/orders/domain/ports"
	paymentsModels "Encargalo.app-api.go/internal/payments/domain/models"
	paymentsPorts "Encargalo.app-api.go/internal/payments/domain/ports"
)

const paymentProofAlertWorkerInterval = 30 * time.Second

// PaymentProofAlertWorker republica payment_proof_pending cada 3 minutos
// mientras el comprobante de una orden siga en pending_review.
type PaymentProofAlertWorker struct {
	reviews paymentsPorts.PaymentReviewsRepo
	orders  ports.OrdersRepo
	stream  ports.RedisStream
}

func NewPaymentProofAlertWorker(reviews paymentsPorts.PaymentReviewsRepo, orders ports.OrdersRepo, stream ports.RedisStream) *PaymentProofAlertWorker {
	return &PaymentProofAlertWorker{reviews: reviews, orders: orders, stream: stream}
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
	if w.stream == nil || w.reviews == nil {
		return
	}
	now := time.Now()
	alertBefore := now.Add(-paymentsModels.PaymentProofAlertInterval)
	list, err := w.reviews.ListPendingForAlert(ctx, paymentsModels.ReferenceTypeOrder, alertBefore)
	if err != nil {
		slog.Error("PaymentProofAlertWorker tick failed", "error", err)
		return
	}
	for i := range list {
		review := list[i]
		ok, err := w.reviews.MarkAlerted(ctx, review.ID, now, alertBefore)
		if err != nil {
			slog.Error("PaymentProofAlertWorker MarkAlerted failed", "reviewID", review.ID, "error", err)
			continue
		}
		if !ok {
			continue
		}
		order, err := w.orders.SearchOrdersByID(ctx, review.ReferenceID)
		if err != nil || order == nil {
			slog.Error("PaymentProofAlertWorker order lookup failed", "reviewID", review.ID, "orderID", review.ReferenceID, "error", err)
			continue
		}
		minutes := max(int(now.Sub(review.CreatedAt)/time.Minute), 0)
		event := models.DataEventPaymentProofPending{}
		event.BuildFrom(*order, review.ID, review.ExpectedAmountBs, review.ProofURL, minutes)
		if err := w.stream.PublisherEvent(ctx, event); err != nil {
			slog.Error("PaymentProofAlertWorker publish failed", "reviewID", review.ID, "error", err)
			continue
		}
		slog.Info("payment_proof_pending published",
			"reviewID", review.ID,
			"orderID", review.ReferenceID,
			"minutesPending", minutes,
		)
	}
}
