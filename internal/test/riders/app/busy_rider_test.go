package app_test

import (
	"context"
	"testing"

	mocksriders "Encargalo.app-api.go/internal/mocks/riders/domain/ports"
	mockstrips "Encargalo.app-api.go/internal/mocks/trips/domain/ports"
	"Encargalo.app-api.go/internal/riders/app"
	riderModels "Encargalo.app-api.go/internal/riders/domain/models"
	ridersPorts "Encargalo.app-api.go/internal/riders/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	tripsModels "Encargalo.app-api.go/internal/trips/domain/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newBusyRiderSvc(t *testing.T) (ridersPorts.RidersApp, *mockstrips.TripsRepo) {
	t.Helper()
	svc, tripsRepo, repo := newTripOfferSvc(t)
	repo.On("SearchActiveOrdersByRiderID", mock.Anything, mock.Anything).Return(nil, errcustom.ErrNotFound)
	repo.On("SearchAvailableOrders", mock.Anything).Return(nil, errcustom.ErrNotFound)
	return svc, tripsRepo
}

// openSearchingTrip deja el viaje en searching para todas las lecturas.
func openSearchingTrip(t *testing.T, svc ridersPorts.RidersApp, tripsRepo *mockstrips.TripsRepo) tripsModels.Trip {
	t.Helper()
	trip := samplePickupTrip(t)
	t.Cleanup(func() { app.StopTripOfferTimerForTest(svc, trip.ID) })
	searching := trip
	searching.Status = tripsModels.TripStatusSearching
	tripsRepo.On("GetTripByID", mock.Anything, trip.ID).Return(&searching, nil).Maybe()
	return trip
}

func acceptOfferedTrip(t *testing.T, svc ridersPorts.RidersApp, tripsRepo *mockstrips.TripsRepo, riderID uuid.UUID, trip tripsModels.Trip) {
	t.Helper()
	tripsRepo.On("AssignTripRider", mock.Anything, trip.ID, riderID, mock.Anything, mock.Anything).Return(true, nil).Once()
	conn := mocksriders.NewRiderConn(t)
	conn.On("Send", mock.Anything).Return().Maybe()
	require.NoError(t, svc.AcceptTrip(context.Background(), riderID, trip.ID, conn))
}

func TestOfferTrip_SkipsRiderWithTripInProgress(t *testing.T) {
	svc, tripsRepo := newBusyRiderSvc(t)
	busyID, busyOffers := registerWatchedRider(t, svc, riderModels.RiderModeTrips)
	freeID, freeOffers := registerWatchedRider(t, svc, riderModels.RiderModeTrips)
	require.NoError(t, svc.UpdateLocation(context.Background(), busyID, 10.4806, -66.9036))
	require.NoError(t, svc.UpdateLocation(context.Background(), freeID, 10.4807, -66.9036))

	first := openSearchingTrip(t, svc, tripsRepo)
	asMatcher(t, svc).OfferTrip(context.Background(), first)
	acceptOfferedTrip(t, svc, tripsRepo, busyID, first)

	second := openSearchingTrip(t, svc, tripsRepo)
	asMatcher(t, svc).OfferTrip(context.Background(), second)

	require.Equal(t, 1, busyOffers())
	require.Equal(t, 2, freeOffers())
}

func TestUpdateLocation_DoesNotOfferOpenTripToBusyRider(t *testing.T) {
	svc, tripsRepo := newBusyRiderSvc(t)
	busyID, busyOffers := registerWatchedRider(t, svc, riderModels.RiderModeTrips)
	require.NoError(t, svc.UpdateLocation(context.Background(), busyID, 10.4806, -66.9036))

	first := openSearchingTrip(t, svc, tripsRepo)
	asMatcher(t, svc).OfferTrip(context.Background(), first)
	acceptOfferedTrip(t, svc, tripsRepo, busyID, first)

	second := openSearchingTrip(t, svc, tripsRepo)
	asMatcher(t, svc).OfferTrip(context.Background(), second)
	nextLat, nextLng := latLngAtRing(t, *second.PickupH3Cell, 1)
	require.NoError(t, svc.UpdateLocation(context.Background(), busyID, nextLat, nextLng))

	require.Equal(t, 1, busyOffers())
}

func TestAcceptTrip_RejectsSecondTripWhileBusy(t *testing.T) {
	svc, tripsRepo := newBusyRiderSvc(t)
	riderID, offers := registerWatchedRider(t, svc, riderModels.RiderModeTrips)
	require.NoError(t, svc.UpdateLocation(context.Background(), riderID, 10.4806, -66.9036))

	first := openSearchingTrip(t, svc, tripsRepo)
	second := openSearchingTrip(t, svc, tripsRepo)
	asMatcher(t, svc).OfferTrip(context.Background(), first)
	asMatcher(t, svc).OfferTrip(context.Background(), second)
	require.Equal(t, 2, offers())
	acceptOfferedTrip(t, svc, tripsRepo, riderID, first)

	err := svc.AcceptTrip(context.Background(), riderID, second.ID, mocksriders.NewRiderConn(t))

	require.ErrorIs(t, err, errcustom.ErrRiderBusy)
	tripsRepo.AssertNotCalled(t, "AssignTripRider", mock.Anything, second.ID, riderID, mock.Anything, mock.Anything)
}

func TestAcceptTrip_DatabaseRejectsBusyRider(t *testing.T) {
	svc, tripsRepo := newBusyRiderSvc(t)
	riderID, _ := registerWatchedRider(t, svc, riderModels.RiderModeTrips)
	require.NoError(t, svc.UpdateLocation(context.Background(), riderID, 10.4806, -66.9036))

	trip := openSearchingTrip(t, svc, tripsRepo)
	asMatcher(t, svc).OfferTrip(context.Background(), trip)
	// Sin estado en memoria (p. ej. tras reiniciar): la BD rechaza y el viaje sigue en searching.
	tripsRepo.On("AssignTripRider", mock.Anything, trip.ID, riderID, mock.Anything, mock.Anything).Return(false, nil)
	conn := mocksriders.NewRiderConn(t)

	err := svc.AcceptTrip(context.Background(), riderID, trip.ID, conn)

	require.ErrorIs(t, err, errcustom.ErrRiderBusy)
	conn.AssertNotCalled(t, "Send", mock.Anything)
}
