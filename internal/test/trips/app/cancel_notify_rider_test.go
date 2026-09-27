package app_test

import (
	"context"
	"testing"

	mockstrips "Encargalo.app-api.go/internal/mocks/trips/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/trips/app"
	"Encargalo.app-api.go/internal/trips/domain/dtos"
	"Encargalo.app-api.go/internal/trips/domain/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCancelTrip_NotifiesRidersWithCancelledTrip(t *testing.T) {
	d := newTripsDeps(t)
	notifier := mockstrips.NewTripRiderNotifier(t)
	svc := app.NewTripsApp(d.repo, d.matcher, d.bcv, d.quotes, d.upload, d.reviews, d.hub, d.riders, d.balance, nil, nil, notifier)

	tripID, customerID, riderID := uuid.New(), uuid.New(), uuid.New()
	d.repo.On("GetTripByID", mock.Anything, tripID).Return(&models.Trip{
		ID: tripID, CustomerID: customerID, RiderID: &riderID, Status: models.TripStatusAccepted,
	}, nil)
	d.repo.On("UpdateTripStatus", mock.Anything, tripID, models.TripStatusAccepted, models.TripStatusCancelled).Return(true, nil)
	d.riders.On("GetByRiderID", mock.Anything, riderID).Return(dtos.RiderPublicInfo{}, nil)
	mockBcvRate(d.bcv, 40)
	notifier.On("NotifyTripCancelled", mock.Anything, mock.MatchedBy(func(trip models.Trip) bool {
		return trip.ID == tripID && trip.Status == models.TripStatusCancelled && trip.RiderID != nil && *trip.RiderID == riderID
	})).Return().Once()

	resp, err := svc.CancelTrip(context.Background(), tripID, customerID)

	require.NoError(t, err)
	require.Equal(t, string(models.TripStatusCancelled), resp.Status)
}

func TestCancelTrip_DoesNotNotifyRidersWhenNotCancellable(t *testing.T) {
	d := newTripsDeps(t)
	notifier := mockstrips.NewTripRiderNotifier(t)
	svc := app.NewTripsApp(d.repo, d.matcher, d.bcv, d.quotes, d.upload, d.reviews, d.hub, d.riders, d.balance, nil, nil, notifier)

	tripID, customerID := uuid.New(), uuid.New()
	d.repo.On("GetTripByID", mock.Anything, tripID).Return(&models.Trip{
		ID: tripID, CustomerID: customerID, Status: models.TripStatusCompleted,
	}, nil)

	_, err := svc.CancelTrip(context.Background(), tripID, customerID)

	require.ErrorIs(t, err, errcustom.ErrTripNotCancellable)
	notifier.AssertNotCalled(t, "NotifyTripCancelled", mock.Anything, mock.Anything)
}
