package app

import (
	"context"
	"log/slog"
	"time"

	paymentsModels "Encargalo.app-api.go/internal/payments/domain/models"
	paymentsPorts "Encargalo.app-api.go/internal/payments/domain/ports"
	"Encargalo.app-api.go/internal/pkg/money"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/shipments/domain/models"
	"Encargalo.app-api.go/internal/shipments/domain/ports"
	usersModels "Encargalo.app-api.go/internal/users/domain/models"
	"github.com/google/uuid"
)

// ShipmentPaymentSubject conecta un envío al módulo de pagos. El matching no
// arranca hasta OnApproved u OnOverpaid. El crédito de balance lo resuelve
// payments/app antes de llamar al subject.
type ShipmentPaymentSubject struct {
	repo      ports.ShipmentsRepo
	matcher   ports.ShipmentRiderMatcher
	customers ports.CustomerShipmentNotifier
	balance   paymentsPorts.CustomerBalanceCreditor
	reviews   paymentsPorts.PaymentReviewsRepo
}

func NewShipmentPaymentSubject(repo ports.ShipmentsRepo, matcher ports.ShipmentRiderMatcher, customers ports.CustomerShipmentNotifier, balance paymentsPorts.CustomerBalanceCreditor, reviews paymentsPorts.PaymentReviewsRepo) *ShipmentPaymentSubject {
	return &ShipmentPaymentSubject{repo: repo, matcher: matcher, customers: customers, balance: balance, reviews: reviews}
}

func (s *ShipmentPaymentSubject) ExpectedAmountBs(ctx context.Context, referenceID uuid.UUID) (float64, error) {
	shipment, err := s.loadInPaymentReview(ctx, referenceID)
	if err != nil {
		return 0, err
	}
	amount, ok := expectedAmountBs(shipment)
	if !ok {
		return 0, errcustom.ErrUnexpectedError
	}
	return amount, nil
}

func (s *ShipmentPaymentSubject) CustomerID(ctx context.Context, referenceID uuid.UUID) (uuid.UUID, error) {
	shipment, err := s.repo.GetShipmentByID(ctx, referenceID)
	if err != nil {
		return uuid.Nil, err
	}
	return shipment.CustomerID, nil
}

func (s *ShipmentPaymentSubject) OnApproved(ctx context.Context, referenceID, _ uuid.UUID) error {
	return s.startMatching(ctx, referenceID)
}

func (s *ShipmentPaymentSubject) OnOverpaid(ctx context.Context, referenceID, _ uuid.UUID, _ float64, _ paymentsPorts.BalanceCredit) error {
	return s.startMatching(ctx, referenceID)
}

func (s *ShipmentPaymentSubject) OnRejected(ctx context.Context, referenceID, reviewID uuid.UUID, reason string, credit paymentsPorts.BalanceCredit) error {
	shipment, err := s.loadInPaymentReview(ctx, referenceID)
	if err != nil {
		return err
	}
	ok, err := s.repo.UpdateShipmentStatus(ctx, shipment.ID, models.ShipmentStatusPaymentReview, models.ShipmentStatusCancelled)
	if err != nil {
		return err
	}
	if !ok {
		return errcustom.ErrInvalidStatus
	}
	shipment.Status = models.ShipmentStatusCancelled
	saveCancelReason(ctx, shipment, cancelReasonFromRejection(reason), s.repo.SaveShipmentCancelReason)
	s.syncRejectedPayment(ctx, shipment, reviewID, credit)
	s.refundBalancePaid(ctx, shipment)
	s.notify(ctx, shipment)
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

// syncRejectedPayment congela en el envío lo que el cliente necesita saber de
// su dinero tras el rechazo. Un fallo aquí no revierte la cancelación.
func (s *ShipmentPaymentSubject) syncRejectedPayment(ctx context.Context, shipment *models.Shipment, reviewID uuid.UUID, credit paymentsPorts.BalanceCredit) {
	if s.reviews == nil {
		return
	}
	review, err := s.reviews.GetByID(ctx, reviewID)
	if err != nil {
		slog.Error("shipment rejected payment: read review", "shipmentID", shipment.ID, "reviewID", reviewID, "error", err)
		return
	}

	payment := &models.ShipmentPayment{
		Status:           string(review.Status),
		ExpectedAmountBs: review.ExpectedAmountBs,
		ObservedAmountBs: review.ObservedAmountBs,
		DifferenceBs:     review.DifferenceBs,
		CreditedBs:       review.CreditedBs,
	}
	if review.Status == paymentsModels.ReviewRejectedUnderpaid {
		payment.RejectionReason = review.RejectionReason
	}
	applyBalanceCredit(payment, credit, review.BcvRate)

	shipment.Payment = payment
	if err := s.repo.SaveShipmentPayment(ctx, shipment.ID, payment); err != nil {
		slog.Error("shipment rejected payment: save", "shipmentID", shipment.ID, "error", err)
	}
}

// applyBalanceCredit habilita la devolución a la cuenta del cliente con las
// mismas reglas que trips/órdenes: umbral mínimo y ventana desde el movimiento.
func applyBalanceCredit(payment *models.ShipmentPayment, credit paymentsPorts.BalanceCredit, bcvRate float64) {
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

// refundBalancePaid devuelve el saldo del envío; el de Pago Móvil lo acredita payments.
func (s *ShipmentPaymentSubject) refundBalancePaid(ctx context.Context, shipment *models.Shipment) {
	if s.balance == nil || shipment.BalancePaidBs == nil || *shipment.BalancePaidBs <= 0 {
		return
	}
	if err := s.balance.RefundBalancePayment(ctx, shipment.CustomerID, shipment.ID, models.ReferenceTypeShipment, *shipment.BalancePaidBs); err != nil {
		slog.Error("shipment payment rejected refund balance", "shipmentID", shipment.ID, "error", err)
	}
}

func (s *ShipmentPaymentSubject) notify(ctx context.Context, shipment *models.Shipment) {
	if s.customers == nil || shipment == nil {
		return
	}
	s.customers.NotifyShipment(ctx, *shipment)
}

func (s *ShipmentPaymentSubject) EnrichStaffReview(ctx context.Context, review paymentsModels.PaymentReview) (paymentsModels.PaymentReviewBroadcast, error) {
	shipment, err := s.repo.GetShipmentByID(ctx, review.ReferenceID)
	if err != nil {
		return paymentsModels.PaymentReviewBroadcast{}, err
	}
	return broadcastFromShipment(&review, shipment), nil
}

func (s *ShipmentPaymentSubject) startMatching(ctx context.Context, referenceID uuid.UUID) error {
	shipment, err := s.loadInPaymentReview(ctx, referenceID)
	if err != nil {
		return err
	}
	ok, err := s.repo.UpdateShipmentStatus(ctx, shipment.ID, models.ShipmentStatusPaymentReview, models.ShipmentStatusSearching)
	if err != nil {
		return err
	}
	if !ok {
		return errcustom.ErrInvalidStatus
	}
	shipment.Status = models.ShipmentStatusSearching
	if s.matcher != nil {
		s.matcher.OfferShipment(ctx, *shipment)
	}
	s.notify(ctx, shipment)
	return nil
}

func (s *ShipmentPaymentSubject) loadInPaymentReview(ctx context.Context, referenceID uuid.UUID) (*models.Shipment, error) {
	shipment, err := s.repo.GetShipmentByID(ctx, referenceID)
	if err != nil {
		return nil, err
	}
	if shipment.Status != models.ShipmentStatusPaymentReview {
		return nil, errcustom.ErrInvalidStatus
	}
	return shipment, nil
}

func expectedAmountBs(shipment *models.Shipment) (float64, bool) {
	if shipment == nil || shipment.Fare == nil || shipment.BcvRate == nil || *shipment.Fare <= 0 || *shipment.BcvRate <= 0 {
		return 0, false
	}
	return money.RoundBs(*shipment.Fare * *shipment.BcvRate), true
}

func broadcastFromShipment(review *paymentsModels.PaymentReview, shipment *models.Shipment) paymentsModels.PaymentReviewBroadcast {
	b := paymentsModels.PaymentReviewBroadcast{
		Type:             "payment_review_update",
		ReviewID:         review.ID,
		ReferenceID:      shipment.ID,
		ReferenceType:    models.ReferenceTypeShipment,
		CustomerID:       shipment.CustomerID,
		ProofURL:         review.ProofURL,
		ExpectedAmountBs: review.ExpectedAmountBs,
		ObservedAmountBs: review.ObservedAmountBs,
		DifferenceBs:     review.DifferenceBs,
		Status:           review.Status,
		RejectionReason:  review.RejectionReason,
		CreditedBs:       review.CreditedBs,
		CreatedAt:        review.CreatedAt,
	}
	if shipment.Fare != nil {
		b.TotalUSD = *shipment.Fare
	}
	b.PagoMovilAmountBs = review.ExpectedAmountBs
	return b
}

var _ paymentsPorts.PaymentSubject = (*ShipmentPaymentSubject)(nil)
var _ paymentsPorts.StaffReviewEnricher = (*ShipmentPaymentSubject)(nil)
