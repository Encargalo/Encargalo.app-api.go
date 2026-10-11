package app

import (
	"context"
	"log/slog"
	"time"

	paymentsModels "Encargalo.app-api.go/internal/payments/domain/models"
	paymentsPorts "Encargalo.app-api.go/internal/payments/domain/ports"
	"Encargalo.app-api.go/internal/trips/domain/models"
	"Encargalo.app-api.go/internal/trips/domain/ports"
)

const tripPaymentProofAlertWorkerInterval = 30 * time.Second

// TripPaymentProofAlertWorker republica payment_proof_pending cada 3 minutos
// mientras el comprobante de un viaje siga en pending_review.
type TripPaymentProofAlertWorker struct {
	reviews paymentsPorts.PaymentReviewsRepo
	events  ports.TripEventPublisher
}

func NewTripPaymentProofAlertWorker(reviews paymentsPorts.PaymentReviewsRepo, events ports.TripEventPublisher) *TripPaymentProofAlertWorker {
	return &TripPaymentProofAlertWorker{reviews: reviews, events: events}
}

func (w *TripPaymentProofAlertWorker) Start(ctx context.Context) {
	go w.loop(ctx)
}

func (w *TripPaymentProofAlertWorker) loop(ctx context.Context) {
	ticker := time.NewTicker(tripPaymentProofAlertWorkerInterval)
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

func (w *TripPaymentProofAlertWorker) Tick(ctx context.Context) {
	if w.events == nil || w.reviews == nil {
		return
	}
	now := time.Now()
	alertBefore := now.Add(-paymentsModels.PaymentProofAlertInterval)
	list, err := w.reviews.ListPendingForAlert(ctx, models.ReferenceTypeTrip, alertBefore)
	if err != nil {
		slog.Error("TripPaymentProofAlertWorker tick failed", "error", err)
		return
	}
	for i := range list {
		review := list[i]
		ok, err := w.reviews.MarkAlerted(ctx, review.ID, now, alertBefore)
		if err != nil {
			slog.Error("TripPaymentProofAlertWorker MarkAlerted failed", "reviewID", review.ID, "error", err)
			continue
		}
		if !ok {
			continue
		}
		minutes := max(int(now.Sub(review.CreatedAt)/time.Minute), 0)
		event := models.DataEventTripPaymentProofPending{}
		event.BuildFrom(review.ReferenceID, review.ID, review.ExpectedAmountBs, review.ProofURL, minutes)
		if err := w.events.PublisherEvent(ctx, event); err != nil {
			slog.Error("TripPaymentProofAlertWorker publish failed", "reviewID", review.ID, "error", err)
			continue
		}
		slog.Info("trip payment_proof_pending published", "reviewID", review.ID, "tripID", review.ReferenceID, "minutesPending", minutes)
	}
}
