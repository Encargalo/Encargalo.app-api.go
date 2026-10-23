package app

import (
	"context"
	"log/slog"
	"time"

	paymentsModels "Encargalo.app-api.go/internal/payments/domain/models"
	paymentsPorts "Encargalo.app-api.go/internal/payments/domain/ports"
	"Encargalo.app-api.go/internal/pkg/money"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/trips/domain/models"
	"Encargalo.app-api.go/internal/trips/domain/ports"
	usersModels "Encargalo.app-api.go/internal/users/domain/models"
	"github.com/google/uuid"
)

// TripPaymentSubject conecta un viaje al módulo de pagos. El matching no
// arranca hasta OnApproved u OnOverpaid. El crédito de balance lo resuelve
// payments/app antes de llamar al subject.
type TripPaymentSubject struct {
	repo      ports.TripsRepo
	matcher   ports.TripRiderMatcher
	customers ports.CustomerTripNotifier
	balance   paymentsPorts.CustomerBalanceCreditor
	reviews   paymentsPorts.PaymentReviewsRepo
}

func NewTripPaymentSubject(repo ports.TripsRepo, matcher ports.TripRiderMatcher, customers ports.CustomerTripNotifier, balance paymentsPorts.CustomerBalanceCreditor, reviews paymentsPorts.PaymentReviewsRepo) *TripPaymentSubject {
	return &TripPaymentSubject{repo: repo, matcher: matcher, customers: customers, balance: balance, reviews: reviews}
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

func (s *TripPaymentSubject) OnRejected(ctx context.Context, referenceID, reviewID uuid.UUID, reason string, credit paymentsPorts.BalanceCredit) error {
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
	trip.Status = models.TripStatusCancelled
	saveCancelReason(ctx, trip, cancelReasonFromRejection(reason), s.repo.SaveTripCancelReason)
	s.syncRejectedPayment(ctx, trip, reviewID, credit)
	s.refundBalancePaid(ctx, trip)
	s.notify(ctx, trip)
	return nil
}

// cancelReasonFromRejection traduce el motivo que manda payments. RejectUnderpaid
// manda el centinela "underpaid"; RejectInvalid manda el texto libre del staff,
// que no se expone: cualquier otro valor cae en payment_rejected.
func cancelReasonFromRejection(reason string) models.CancelReason {
	if reason == paymentsModels.RejectionReasonUnderpaid {
		return models.CancelReasonPaymentUnderpaid
	}
	return models.CancelReasonPaymentRejected
}

// syncRejectedPayment congela en el viaje lo que el cliente necesita saber de su
// dinero tras el rechazo. Un fallo aquí no revierte la cancelación.
func (s *TripPaymentSubject) syncRejectedPayment(ctx context.Context, trip *models.Trip, reviewID uuid.UUID, credit paymentsPorts.BalanceCredit) {
	if s.reviews == nil {
		return
	}
	review, err := s.reviews.GetByID(ctx, reviewID)
	if err != nil {
		slog.Error("trip rejected payment: read review", "tripID", trip.ID, "reviewID", reviewID, "error", err)
		return
	}

	payment := &models.TripPayment{
		Status:           string(review.Status),
		ExpectedAmountBs: review.ExpectedAmountBs,
		ObservedAmountBs: review.ObservedAmountBs,
		DifferenceBs:     review.DifferenceBs,
		CreditedBs:       review.CreditedBs,
	}
	// El texto de rejected_invalid lo escribe el staff y puede nombrar la
	// referencia del comprobante duplicado: solo sale el de monto insuficiente.
	if review.Status == paymentsModels.ReviewRejectedUnderpaid {
		payment.RejectionReason = review.RejectionReason
	}
	applyBalanceCredit(payment, credit, review.BcvRate)

	trip.Payment = payment
	if err := s.repo.SaveTripPayment(ctx, trip.ID, payment); err != nil {
		slog.Error("trip rejected payment: save", "tripID", trip.ID, "error", err)
	}
}

// applyBalanceCredit habilita la devolución a la cuenta del cliente con las
// mismas reglas que órdenes: umbral mínimo y ventana desde el movimiento.
func applyBalanceCredit(payment *models.TripPayment, credit paymentsPorts.BalanceCredit, bcvRate float64) {
	if credit.TxID == uuid.Nil {
		return
	}
	txID := credit.TxID
	payment.TransactionID = &txID
	if !usersModels.MeetsRefundMin(credit.RefundablePMBs, bcvRate) {
		return
	}
	refundable := credit.RefundablePMBs
	until := credit.CreatedAt.Add(usersModels.RefundRequestWindow)
	untilStr := until.Format(time.RFC3339)
	payment.RefundablePMBs = &refundable
	payment.RefundEligibleUntil = &untilStr
	payment.CanRequestRefund = !time.Now().After(until)
}

// refundBalancePaid devuelve la porción de saldo de un viaje mixto; la de Pago Móvil la acredita payments.
func (s *TripPaymentSubject) refundBalancePaid(ctx context.Context, trip *models.Trip) {
	if s.balance == nil || trip.BalancePaidBs == nil || *trip.BalancePaidBs <= 0 {
		return
	}
	if err := s.balance.RefundBalancePayment(ctx, trip.CustomerID, trip.ID, models.ReferenceTypeTrip, *trip.BalancePaidBs); err != nil {
		slog.Error("trip payment rejected refund balance", "tripID", trip.ID, "error", err)
	}
}

func (s *TripPaymentSubject) notify(ctx context.Context, trip *models.Trip) {
	if s.customers == nil || trip == nil {
		return
	}
	s.customers.NotifyTrip(ctx, *trip)
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
	s.notify(ctx, trip)
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
