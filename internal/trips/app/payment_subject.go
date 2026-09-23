package app

import (
	"context"

	paymentsModels "Encargalo.app-api.go/internal/payments/domain/models"
	paymentsPorts "Encargalo.app-api.go/internal/payments/domain/ports"
	"Encargalo.app-api.go/internal/pkg/money"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/trips/domain/models"
	"Encargalo.app-api.go/internal/trips/domain/ports"
	"github.com/google/uuid"
)

// TripPaymentSubject conecta un viaje al módulo de pagos. El matching no
// arranca hasta OnApproved u OnOverpaid. El crédito de balance lo resuelve
// payments/app antes de llamar al subject.
type TripPaymentSubject struct {
	repo    ports.TripsRepo
	matcher ports.TripRiderMatcher
}

func NewTripPaymentSubject(repo ports.TripsRepo, matcher ports.TripRiderMatcher) *TripPaymentSubject {
	return &TripPaymentSubject{repo: repo, matcher: matcher}
}

func (s *TripPaymentSubject) ExpectedAmountBs(ctx context.Context, referenceID uuid.UUID) (float64, error) {
	trip, err := s.loadInPaymentReview(ctx, referenceID)
	if err != nil {
		return 0, err
	}
	amount, ok := expectedAmountBs(trip)
	if !ok {
		return 0, errcustom.ErrUnexpectedError
	}
	return amount, nil
}

func (s *TripPaymentSubject) CustomerID(ctx context.Context, referenceID uuid.UUID) (uuid.UUID, error) {
	trip, err := s.repo.GetTripByID(ctx, referenceID)
	if err != nil {
		return uuid.Nil, err
	}
	return trip.CustomerID, nil
}

func (s *TripPaymentSubject) OnApproved(ctx context.Context, referenceID, _ uuid.UUID) error {
	return s.startMatching(ctx, referenceID)
}

func (s *TripPaymentSubject) OnOverpaid(ctx context.Context, referenceID, _ uuid.UUID, _ float64, _ paymentsPorts.BalanceCredit) error {
	return s.startMatching(ctx, referenceID)
}

func (s *TripPaymentSubject) OnRejected(ctx context.Context, referenceID, _ uuid.UUID, _ string, _ paymentsPorts.BalanceCredit) error {
	trip, err := s.loadInPaymentReview(ctx, referenceID)
	if err != nil {
		return err
	}
	ok, err := s.repo.UpdateTripStatus(ctx, trip.ID, models.TripStatusPaymentReview, models.TripStatusCancelled)
	if err != nil {
		return err
	}
	if !ok {
		return errcustom.ErrInvalidStatus
	}
	return nil
}

func (s *TripPaymentSubject) EnrichStaffReview(ctx context.Context, review paymentsModels.PaymentReview) (paymentsModels.PaymentReviewBroadcast, error) {
	trip, err := s.repo.GetTripByID(ctx, review.ReferenceID)
	if err != nil {
		return paymentsModels.PaymentReviewBroadcast{}, err
	}
	return broadcastFromTrip(&review, trip), nil
}

func (s *TripPaymentSubject) startMatching(ctx context.Context, referenceID uuid.UUID) error {
	trip, err := s.loadInPaymentReview(ctx, referenceID)
	if err != nil {
		return err
	}
	ok, err := s.repo.UpdateTripStatus(ctx, trip.ID, models.TripStatusPaymentReview, models.TripStatusSearching)
	if err != nil {
		return err
	}
	if !ok {
		return errcustom.ErrInvalidStatus
	}
	trip.Status = models.TripStatusSearching
	if s.matcher != nil {
		s.matcher.OfferTrip(ctx, *trip)
	}
	return nil
}

func (s *TripPaymentSubject) loadInPaymentReview(ctx context.Context, referenceID uuid.UUID) (*models.Trip, error) {
	trip, err := s.repo.GetTripByID(ctx, referenceID)
	if err != nil {
		return nil, err
	}
	if trip.Status != models.TripStatusPaymentReview {
		return nil, errcustom.ErrInvalidStatus
	}
	return trip, nil
}

func expectedAmountBs(trip *models.Trip) (float64, bool) {
	if trip == nil || trip.Fare == nil || trip.BcvRate == nil || *trip.Fare <= 0 || *trip.BcvRate <= 0 {
		return 0, false
	}
	return money.RoundBs(*trip.Fare * *trip.BcvRate), true
}

func broadcastFromTrip(review *paymentsModels.PaymentReview, trip *models.Trip) paymentsModels.PaymentReviewBroadcast {
	b := paymentsModels.PaymentReviewBroadcast{
		Type:             "payment_review_update",
		ReviewID:         review.ID,
		ReferenceID:      trip.ID,
		ReferenceType:    models.ReferenceTypeTrip,
		CustomerID:       trip.CustomerID,
		ProofURL:         review.ProofURL,
		ExpectedAmountBs: review.ExpectedAmountBs,
		ObservedAmountBs: review.ObservedAmountBs,
		DifferenceBs:     review.DifferenceBs,
		Status:           review.Status,
		RejectionReason:  review.RejectionReason,
		CreditedBs:       review.CreditedBs,
		CreatedAt:        review.CreatedAt,
	}
	if trip.Fare != nil {
		b.TotalUSD = *trip.Fare
	}
	b.PagoMovilAmountBs = review.ExpectedAmountBs
	return b
}

var _ paymentsPorts.PaymentSubject = (*TripPaymentSubject)(nil)
var _ paymentsPorts.StaffReviewEnricher = (*TripPaymentSubject)(nil)
