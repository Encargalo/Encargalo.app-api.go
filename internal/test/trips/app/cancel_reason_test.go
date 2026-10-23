package app_test

import (
	"context"
	"testing"
	"time"

	mockpayments "Encargalo.app-api.go/internal/mocks/payments/domain/ports"
	mockstrips "Encargalo.app-api.go/internal/mocks/trips/domain/ports"
	paymentsModels "Encargalo.app-api.go/internal/payments/domain/models"
	paymentsPorts "Encargalo.app-api.go/internal/payments/domain/ports"
	"Encargalo.app-api.go/internal/trips/app"
	"Encargalo.app-api.go/internal/trips/domain/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func rejectedReview(id uuid.UUID, status paymentsModels.PaymentReviewStatus, reason string) *paymentsModels.PaymentReview {
	observed := 150.0
	diff := -50.0
	credited := 150.0
	return &paymentsModels.PaymentReview{
		ID:               id,
		ExpectedAmountBs: 200.0,
		ObservedAmountBs: &observed,
		DifferenceBs:     &diff,
		CreditedBs:       &credited,
		BcvRate:          40.0,
		Status:           status,
		RejectionReason:  &reason,
	}
}

func TestOnRejected_UnderpaidExposesReasonAndAmounts(t *testing.T) {
	repo := mockstrips.NewTripsRepo(t)
	reviews := mockpayments.NewPaymentReviewsRepo(t)
	subject := app.NewTripPaymentSubject(repo, nil, nil, nil, reviews)
	tripID, reviewID := uuid.New(), uuid.New()
	trip := paymentReviewTrip(tripID)
	txID := uuid.New()

	repo.On("GetTripByID", mock.Anything, tripID).Return(trip, nil)
	repo.On("UpdateTripStatus", mock.Anything, tripID, models.TripStatusPaymentReview, models.TripStatusCancelled).Return(true, nil)
	repo.On("SaveTripCancelReason", mock.Anything, tripID, models.CancelReasonPaymentUnderpaid).Return(nil).Once()
	repo.On("SaveTripPayment", mock.Anything, tripID, mock.Anything).Return(nil).Once()
	reviews.On("GetByID", mock.Anything, reviewID).
		Return(rejectedReview(reviewID, paymentsModels.ReviewRejectedUnderpaid, "pagó de menos"), nil)

	credit := paymentsPorts.BalanceCredit{TxID: txID, RefundablePMBs: 150.0, CreatedAt: time.Now()}
	err := subject.OnRejected(context.Background(), tripID, reviewID, paymentsModels.RejectionReasonUnderpaid, credit)
	require.NoError(t, err)

	resp := trip.ToCustomerDTO()
	require.Equal(t, string(models.CancelReasonPaymentUnderpaid), resp.CancelReason)
	require.NotNil(t, resp.Payment)
	require.Equal(t, string(paymentsModels.ReviewRejectedUnderpaid), resp.Payment.Status)
	require.InDelta(t, 200.0, resp.Payment.ExpectedAmountBs, 0.001)
	require.InDelta(t, 150.0, *resp.Payment.ObservedAmountBs, 0.001)
	require.InDelta(t, -50.0, *resp.Payment.DifferenceBs, 0.001)
	require.Equal(t, "pagó de menos", *resp.Payment.RejectionReason)
	require.Equal(t, txID, *resp.Payment.TransactionID)
	require.True(t, resp.Payment.CanRequestRefund)
}

// El motivo que escribe el staff en rejected_invalid puede nombrar la
// referencia del comprobante duplicado: nunca debe llegar al cliente.
func TestOnRejected_InvalidHidesStaffText(t *testing.T) {
	repo := mockstrips.NewTripsRepo(t)
	reviews := mockpayments.NewPaymentReviewsRepo(t)
	subject := app.NewTripPaymentSubject(repo, nil, nil, nil, reviews)
	tripID, reviewID := uuid.New(), uuid.New()
	trip := paymentReviewTrip(tripID)
	staffText := "comprobante duplicado (referencia 4821)"

	repo.On("GetTripByID", mock.Anything, tripID).Return(trip, nil)
	repo.On("UpdateTripStatus", mock.Anything, tripID, models.TripStatusPaymentReview, models.TripStatusCancelled).Return(true, nil)
	repo.On("SaveTripCancelReason", mock.Anything, tripID, models.CancelReasonPaymentRejected).Return(nil).Once()
	repo.On("SaveTripPayment", mock.Anything, tripID, mock.Anything).Return(nil).Once()
	reviews.On("GetByID", mock.Anything, reviewID).
		Return(rejectedReview(reviewID, paymentsModels.ReviewRejectedInvalid, staffText), nil)

	err := subject.OnRejected(context.Background(), tripID, reviewID, staffText, paymentsPorts.BalanceCredit{})
	require.NoError(t, err)

	resp := trip.ToCustomerDTO()
	require.Equal(t, string(models.CancelReasonPaymentRejected), resp.CancelReason)
	require.Nil(t, resp.Payment.RejectionReason)
	require.Nil(t, resp.Payment.TransactionID)
}

func TestCancelTrip_SetsCustomerCancelledReason(t *testing.T) {
	svc, repo, _, bcv, _ := newTripsApp(t)
	tripID, customerID := uuid.New(), uuid.New()

	repo.On("GetTripByID", mock.Anything, tripID).Return(&models.Trip{ID: tripID, CustomerID: customerID, Status: models.TripStatusSearching}, nil)
	repo.On("UpdateTripStatus", mock.Anything, tripID, models.TripStatusSearching, models.TripStatusCancelled).Return(true, nil)
	repo.On("SaveTripCancelReason", mock.Anything, tripID, models.CancelReasonCustomerCancelled).Return(nil).Once()
	mockBcvRate(bcv, 40)

	resp, err := svc.CancelTrip(context.Background(), tripID, customerID)

	require.NoError(t, err)
	require.Equal(t, string(models.CancelReasonCustomerCancelled), resp.CancelReason)
}

// El rider no debe ver por qué se canceló ni el estado del pago del cliente.
func TestRiderDTO_OmitsCancelReasonAndPayment(t *testing.T) {
	reason := models.CancelReasonPaymentRejected
	trip := &models.Trip{
		ID:           uuid.New(),
		Status:       models.TripStatusCancelled,
		CancelReason: &reason,
		Payment:      &models.TripPayment{Status: string(paymentsModels.ReviewRejectedInvalid)},
	}

	require.Empty(t, trip.ToDTO().CancelReason)
	require.Nil(t, trip.ToDTO().Payment)
	require.Equal(t, string(reason), trip.ToCustomerDTO().CancelReason)
}
