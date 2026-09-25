package app_test

import (
	"context"
	"testing"

	mockpayments "Encargalo.app-api.go/internal/mocks/payments/domain/ports"
	mockstrips "Encargalo.app-api.go/internal/mocks/trips/domain/ports"
	paymentsModels "Encargalo.app-api.go/internal/payments/domain/models"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/trips/app"
	"Encargalo.app-api.go/internal/trips/domain/models"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCancelTrip_PaymentReviewIsRejected(t *testing.T) {
	d := newTripsDeps(t)
	tripID, customerID := uuid.New(), uuid.New()

	d.repo.On("GetTripByID", mock.Anything, tripID).Return(&models.Trip{
		ID: tripID, CustomerID: customerID, Status: models.TripStatusPaymentReview,
	}, nil)

	_, err := d.svc.CancelTrip(context.Background(), tripID, customerID)

	require.ErrorIs(t, err, errcustom.ErrTripPaymentReviewPending)
	httpErr, ok := errcustom.ToHTTPError(err).(*echo.HTTPError)
	require.True(t, ok)
	require.Equal(t, 422, httpErr.Code)
	d.repo.AssertNotCalled(t, "UpdateTripStatus", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestTripStaffSnapshot_SkipsCancelledTrips(t *testing.T) {
	repo := mockstrips.NewTripsRepo(t)
	reviews := mockpayments.NewPaymentReviewsRepo(t)
	openID := uuid.New()
	cancelledID := uuid.New()

	reviews.On("ListActiveForStaff", mock.Anything).Return([]paymentsModels.PaymentReview{
		{ID: uuid.New(), ReferenceID: openID, ReferenceType: models.ReferenceTypeTrip},
		{ID: uuid.New(), ReferenceID: cancelledID, ReferenceType: models.ReferenceTypeTrip},
	}, nil)
	repo.On("GetTripByID", mock.Anything, openID).Return(&models.Trip{
		ID: openID, Status: models.TripStatusPaymentReview,
	}, nil)
	repo.On("GetTripByID", mock.Anything, cancelledID).Return(&models.Trip{
		ID: cancelledID, Status: models.TripStatusCancelled,
	}, nil)

	out, err := app.NewTripStaffSnapshot(repo, reviews)(context.Background())

	require.NoError(t, err)
	require.Len(t, out, 1)
	require.Equal(t, openID, out[0].ReferenceID)
}
