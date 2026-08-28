package app_test

import (
	"context"
	"testing"

	mockstrips "Encargalo.app-api.go/internal/mocks/trips/domain/ports"
	"Encargalo.app-api.go/internal/pkg/geo"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/trips/app"
	"Encargalo.app-api.go/internal/trips/domain/dtos"
	"Encargalo.app-api.go/internal/trips/domain/models"
	"Encargalo.app-api.go/internal/trips/domain/ports"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newTripsApp(t *testing.T) (ports.TripsApp, *mockstrips.TripsRepo) {
	t.Helper()
	repo := mockstrips.NewTripsRepo(t)
	return app.NewTripsApp(repo), repo
}

func sampleCreateTripRequest() dtos.CreateTripRequest {
	return dtos.CreateTripRequest{
		PickupAddress:    "Av. Libertador, Caracas",
		PickupLatitude:   10.4806,
		PickupLongitude:  -66.9036,
		DropoffAddress:   "CC Sambil, Caracas",
		DropoffLatitude:  10.4956,
		DropoffLongitude: -66.8517,
	}
}

func TestRequestTrip_ComputesH3CellAndPersists(t *testing.T) {
	svc, repo := newTripsApp(t)
	customerID := uuid.New()
	req := sampleCreateTripRequest()
	wantCell, err := geo.LatLngToCell(req.PickupLatitude, req.PickupLongitude, geo.DriverMatchResolution)
	require.NoError(t, err)

	repo.On("CreateTrip", mock.Anything, mock.MatchedBy(func(trip *models.Trip) bool {
		return trip.CustomerID == customerID &&
			trip.Status == models.TripStatusRequested &&
			trip.PickupH3Cell != nil && *trip.PickupH3Cell == wantCell &&
			trip.DropoffAddress == req.DropoffAddress
	})).Run(func(args mock.Arguments) {
		trip := args.Get(1).(*models.Trip)
		trip.ID = uuid.New()
	}).Return(nil)

	resp, err := svc.RequestTrip(context.Background(), customerID, req)

	require.NoError(t, err)
	require.Equal(t, string(models.TripStatusRequested), resp.Status)
	require.Equal(t, customerID, resp.CustomerID)
	repo.AssertExpectations(t)
}

func TestGetTrip_ForbiddenWhenNotOwner(t *testing.T) {
	svc, repo := newTripsApp(t)
	tripID, ownerID, requesterID := uuid.New(), uuid.New(), uuid.New()

	repo.On("GetTripByID", mock.Anything, tripID).Return(&models.Trip{ID: tripID, CustomerID: ownerID}, nil)

	_, err := svc.GetTrip(context.Background(), tripID, requesterID)

	require.ErrorIs(t, err, errcustom.ErrTripForbidden)
}

func TestGetTrip_NotFound(t *testing.T) {
	svc, repo := newTripsApp(t)
	tripID, requesterID := uuid.New(), uuid.New()

	repo.On("GetTripByID", mock.Anything, tripID).Return(nil, errcustom.ErrNotFound)

	_, err := svc.GetTrip(context.Background(), tripID, requesterID)

	require.ErrorIs(t, err, errcustom.ErrNotFound)
}

func TestGetTripHistory_MapsAllTrips(t *testing.T) {
	svc, repo := newTripsApp(t)
	customerID := uuid.New()

	repo.On("GetTripHistoryByCustomer", mock.Anything, customerID).Return([]models.Trip{
		{ID: uuid.New(), CustomerID: customerID, Status: models.TripStatusCompleted},
		{ID: uuid.New(), CustomerID: customerID, Status: models.TripStatusCancelled},
	}, nil)

	resp, err := svc.GetTripHistory(context.Background(), customerID)

	require.NoError(t, err)
	require.Len(t, resp.Trips, 2)
}

func TestCancelTrip_ForbiddenWhenNotOwner(t *testing.T) {
	svc, repo := newTripsApp(t)
	tripID, ownerID, requesterID := uuid.New(), uuid.New(), uuid.New()

	repo.On("GetTripByID", mock.Anything, tripID).Return(&models.Trip{ID: tripID, CustomerID: ownerID, Status: models.TripStatusRequested}, nil)

	_, err := svc.CancelTrip(context.Background(), tripID, requesterID)

	require.ErrorIs(t, err, errcustom.ErrTripForbidden)
	repo.AssertNotCalled(t, "UpdateTripStatus", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestCancelTrip_RejectsTerminalStatus(t *testing.T) {
	svc, repo := newTripsApp(t)
	tripID, customerID := uuid.New(), uuid.New()

	repo.On("GetTripByID", mock.Anything, tripID).Return(&models.Trip{ID: tripID, CustomerID: customerID, Status: models.TripStatusCompleted}, nil)

	_, err := svc.CancelTrip(context.Background(), tripID, customerID)

	require.ErrorIs(t, err, errcustom.ErrTripNotCancellable)
	repo.AssertNotCalled(t, "UpdateTripStatus", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestCancelTrip_CASLossReturnsNotCancellable(t *testing.T) {
	svc, repo := newTripsApp(t)
	tripID, customerID := uuid.New(), uuid.New()

	repo.On("GetTripByID", mock.Anything, tripID).Return(&models.Trip{ID: tripID, CustomerID: customerID, Status: models.TripStatusRequested}, nil)
	repo.On("UpdateTripStatus", mock.Anything, tripID, models.TripStatusRequested, models.TripStatusCancelled).Return(false, nil)

	_, err := svc.CancelTrip(context.Background(), tripID, customerID)

	require.ErrorIs(t, err, errcustom.ErrTripNotCancellable)
}

func TestCancelTrip_Success(t *testing.T) {
	svc, repo := newTripsApp(t)
	tripID, customerID := uuid.New(), uuid.New()

	repo.On("GetTripByID", mock.Anything, tripID).Return(&models.Trip{ID: tripID, CustomerID: customerID, Status: models.TripStatusSearching}, nil)
	repo.On("UpdateTripStatus", mock.Anything, tripID, models.TripStatusSearching, models.TripStatusCancelled).Return(true, nil)

	resp, err := svc.CancelTrip(context.Background(), tripID, customerID)

	require.NoError(t, err)
	require.Equal(t, string(models.TripStatusCancelled), resp.Status)
}
