package app_test

import (
	"context"
	"math"
	"testing"
	"time"

	mockbcvports "Encargalo.app-api.go/internal/mocks/bcv/domain/ports"
	mocksriders "Encargalo.app-api.go/internal/mocks/riders/domain/ports"
	"Encargalo.app-api.go/internal/pkg/geo"
	"Encargalo.app-api.go/internal/riders/app"
	riderModels "Encargalo.app-api.go/internal/riders/domain/models"
	ridersPorts "Encargalo.app-api.go/internal/riders/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	tripsPorts "Encargalo.app-api.go/internal/trips/domain/ports"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newNearbyMgr(t *testing.T) ridersPorts.RidersApp {
	t.Helper()
	repo := mocksriders.NewRidersRepo(t)
	stream := mocksriders.NewRidersStream(t)
	bcv := mockbcvports.NewBcvApp(t)
	tripsRepo := mocksriders.NewTripsStore(t)
	earnings := mocksriders.NewTripEarningRecorder(t)
	repo.On("SearchActiveOrdersByRiderID", mock.Anything, mock.Anything).Return(nil, errcustom.ErrNotFound).Maybe()
	repo.On("SearchAvailableOrders", mock.Anything).Return(nil, errcustom.ErrNotFound).Maybe()
	return app.New(repo, stream, bcv, tripsRepo, earnings, nil, nil, nil, nil, nil)
}

func asNearby(t *testing.T, svc ridersPorts.RidersApp) tripsPorts.NearbyRidersLookup {
	t.Helper()
	lookup, ok := svc.(tripsPorts.NearbyRidersLookup)
	require.True(t, ok, "appRidersManager debe implementar tripsPorts.NearbyRidersLookup")
	return lookup
}

func placeTripRider(t *testing.T, svc ridersPorts.RidersApp, lat, lng float64) (uuid.UUID, *mocksriders.RiderConn) {
	t.Helper()
	return placeRider(t, svc, lat, lng, riderModels.RiderModeTrips)
}

func placeRider(t *testing.T, svc ridersPorts.RidersApp, lat, lng float64, mode riderModels.RiderMode) (uuid.UUID, *mocksriders.RiderConn) {
	t.Helper()
	id := uuid.New()
	conn := mocksriders.NewRiderConn(t)
	require.NoError(t, svc.Register(context.Background(), id, conn, mode))
	require.NoError(t, svc.UpdateLocation(context.Background(), id, lat, lng))
	return id, conn
}

func TestNearbyRiders_ReturnsRoundedPinsOfAvailableTripsRiders(t *testing.T) {
	svc := newNearbyMgr(t)
	lat, lng := 10.4806, -66.9036
	placeTripRider(t, svc, lat, lng)

	pins, err := asNearby(t, svc).NearbyRiders(lat, lng)

	require.NoError(t, err)
	require.Len(t, pins, 1)
	require.Equal(t, 10.481, pins[0].Latitude)
	require.Equal(t, -66.904, pins[0].Longitude)
}

func TestNearbyRiders_OrdersModeAndBusyAndStaleAreExcluded(t *testing.T) {
	svc := newNearbyMgr(t)
	lat, lng := 10.4806, -66.9036

	placeRider(t, svc, lat, lng, riderModels.RiderModeOrders)

	busyID, _ := placeTripRider(t, svc, lat, lng)
	app.MarkRiderBusyForTest(svc, busyID)

	staleID, _ := placeTripRider(t, svc, lat+0.001, lng)
	app.AgeRiderLocationForTest(svc, staleID, time.Minute)

	pins, err := asNearby(t, svc).NearbyRiders(lat, lng)

	require.NoError(t, err)
	require.Empty(t, pins)
}

func TestNearbyRiders_DropsRidersOutsideTheDiskAndCapsAtThirty(t *testing.T) {
	svc := newNearbyMgr(t)
	lat, lng := 10.4806, -66.9036
	origin, err := geo.LatLngToCell(lat, lng, geo.DriverMatchResolution)
	require.NoError(t, err)
	farLat, farLng := latLngAtRing(t, origin, riderModels.NearbyRiderRings+1)
	placeTripRider(t, svc, farLat, farLng)

	for i := range riderModels.NearbyRiderMaxResults + 1 {
		placeTripRider(t, svc, lat+(float64(i)*0.00001), lng)
	}

	pins, err := asNearby(t, svc).NearbyRiders(lat, lng)

	require.NoError(t, err)
	require.Len(t, pins, riderModels.NearbyRiderMaxResults)
	wantLat := math.Round(lat*riderModels.NearbyCoordDecimals) / riderModels.NearbyCoordDecimals
	require.Equal(t, wantLat, pins[0].Latitude)
}

func TestNearbyRiders_UnregisterRemovesThePin(t *testing.T) {
	svc := newNearbyMgr(t)
	lat, lng := 10.4806, -66.9036
	id, conn := placeTripRider(t, svc, lat, lng)
	conn.On("Close").Return()
	svc.Unregister(id, conn)

	pins, err := asNearby(t, svc).NearbyRiders(lat, lng)
	require.NoError(t, err)
	require.Empty(t, pins)
}
