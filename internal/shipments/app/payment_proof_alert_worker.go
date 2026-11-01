package app

import (
	"context"
	"log/slog"
	"time"

	paymentsModels "Encargalo.app-api.go/internal/payments/domain/models"
	paymentsPorts "Encargalo.app-api.go/internal/payments/domain/ports"
	"Encargalo.app-api.go/internal/shipments/domain/models"
	"Encargalo.app-api.go/internal/shipments/domain/ports"
)

const shipmentPaymentProofAlertWorkerInterval = 30 * time.Second

// ShipmentPaymentProofAlertWorker republica payment_proof_pending cada 3 minutos
// mientras el comprobante de un envío siga en pending_review, igual que
// TripPaymentProofAlertWorker.
type ShipmentPaymentProofAlertWorker struct {
	reviews paymentsPorts.PaymentReviewsRepo
	events  ports.ShipmentEventPublisher
}

func NewShipmentPaymentProofAlertWorker(reviews paymentsPorts.PaymentReviewsRepo, events ports.ShipmentEventPublisher) *ShipmentPaymentProofAlertWorker {
	return &ShipmentPaymentProofAlertWorker{reviews: reviews, events: events}
}

func (w *ShipmentPaymentProofAlertWorker) Start(ctx context.Context) {
	go w.loop(ctx)
}

func (w *ShipmentPaymentProofAlertWorker) loop(ctx context.Context) {
	ticker := time.NewTicker(shipmentPaymentProofAlertWorkerInterval)
	defer ticker.Stop()

	w.Tick(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.Tick(ctx)
		}
	}
}

func (w *ShipmentPaymentProofAlertWorker) Tick(ctx context.Context) {
	if w.events == nil || w.reviews == nil {
		return
	}
	now := time.Now()
	alertBefore := now.Add(-paymentsModels.PaymentProofAlertInterval)
	list, err := w.reviews.ListPendingForAlert(ctx, models.ReferenceTypeShipment, alertBefore)
	if err != nil {
		slog.Error("ShipmentPaymentProofAlertWorker tick failed", "error", err)
		return
	}
	for i := range list {
		review := list[i]
		ok, err := w.reviews.MarkAlerted(ctx, review.ID, now, alertBefore)
		if err != nil {
			slog.Error("ShipmentPaymentProofAlertWorker MarkAlerted failed", "reviewID", review.ID, "error", err)
			continue
		}
		if !ok {
			continue
		}
		minutes := max(int(now.Sub(review.CreatedAt)/time.Minute), 0)
		event := models.DataEventShipmentPaymentProofPending{}
		event.BuildFrom(review.ReferenceID, review.ID, review.ExpectedAmountBs, review.ProofURL, minutes)
		if err := w.events.PublisherEvent(ctx, event); err != nil {
			slog.Error("ShipmentPaymentProofAlertWorker publish failed", "reviewID", review.ID, "error", err)
			continue
		}
		slog.Info("shipment payment_proof_pending published", "reviewID", review.ID, "shipmentID", review.ReferenceID, "minutesPending", minutes)
	}
}
