package app_test

import (
	"context"
	"testing"
	"time"

	"Encargalo.app-api.go/internal/trips/domain/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func pagoMovilTrip(status models.TripStatus) *models.Trip {
	fare := 5.0
	rate := 40.0
	return &models.Trip{
		ID:         uuid.New(),
		CustomerID: uuid.New(),
		Status:     status,
		Fare:       &fare,
		BcvRate:    &rate,
	}
}

func TestCancelTrip_CreditsVerifiedPagoMovilOnce(t *testing.T) {
	d := newTripsDeps(t)
	trip := pagoMovilTrip(models.TripStatusSearching)
	txID := uuid.New()
	created := time.Now()

	d.repo.On("GetTripByID", mock.Anything, trip.ID).Return(trip, nil)
	d.repo.On("UpdateTripStatus", mock.Anything, trip.ID, models.TripStatusSearching, models.TripStatusCancelled).Return(true, nil)
	d.balance.On("CreditOrderRejected", mock.Anything, trip.CustomerID, trip.ID, models.ReferenceTypeTrip, 200.0, 200.0, 40.0).
		Return(txID, created, nil).Once()
	d.repo.On("SaveTripPayment", mock.Anything, trip.ID, mock.MatchedBy(func(p *models.TripPayment) bool {
		return p != nil && p.TransactionID != nil && *p.TransactionID == txID && p.CreditedBs != nil && *p.CreditedBs == 200
	})).Return(nil)
	mockBcvRate(d.bcv, 40)

	resp, err := d.svc.CancelTrip(context.Background(), trip.ID, trip.CustomerID)

	require.NoError(t, err)
	require.Equal(t, string(models.TripStatusCancelled), resp.Status)
	require.NotNil(t, resp.Payment)
	require.Equal(t, txID, *resp.Payment.TransactionID)
	require.NotNil(t, resp.Payment.RefundablePMBs)
	require.True(t, resp.Payment.CanRequestRefund)
	d.balance.AssertNotCalled(t, "RefundBalancePayment", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestCancelTrip_PagoMovilInReviewDoesNotCredit(t *testing.T) {
	d := newTripsDeps(t)
	trip := pagoMovilTrip(models.TripStatusPaymentReview)

	d.repo.On("GetTripByID", mock.Anything, trip.ID).Return(trip, nil)
	d.repo.On("UpdateTripStatus", mock.Anything, trip.ID, models.TripStatusPaymentReview, models.TripStatusCancelled).Return(true, nil)
	mockBcvRate(d.bcv, 40)

	resp, err := d.svc.CancelTrip(context.Background(), trip.ID, trip.CustomerID)

	require.NoError(t, err)
	require.Nil(t, resp.Payment)
	d.balance.AssertNotCalled(t, "CreditOrderRejected", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestCancelTrip_BalanceStillRefundsDebit(t *testing.T) {
	d := newTripsDeps(t)
	trip := pagoMovilTrip(models.TripStatusSearching)
	paid := 200.0
	trip.BalancePaidBs = &paid

	d.repo.On("GetTripByID", mock.Anything, trip.ID).Return(trip, nil)
	d.repo.On("UpdateTripStatus", mock.Anything, trip.ID, models.TripStatusSearching, models.TripStatusCancelled).Return(true, nil)
	d.balance.On("RefundBalancePayment", mock.Anything, trip.CustomerID, trip.ID, models.ReferenceTypeTrip, paid).Return(nil)
	mockBcvRate(d.bcv, 40)

	_, err := d.svc.CancelTrip(context.Background(), trip.ID, trip.CustomerID)

	require.NoError(t, err)
	d.balance.AssertNotCalled(t, "CreditOrderRejected", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}
