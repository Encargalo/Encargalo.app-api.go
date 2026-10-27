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
	ridersRepo "Encargalo.app-api.go/internal/riders/repo"
	"Encargalo.app-api.go/internal/shared/errcustom"
	tripsPorts "Encargalo.app-api.go/internal/trips/domain/ports"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newNearbyMgr(t *testing.T) ridersPorts.RidersApp {
	t.Helper()
	svc, _ := newNearbyMgrWithStore(t)
	return svc
}

// newNearbyMgrWithStore arma el manager contra un Redis en memoria, para que el
// radio, el modo y la frescura los resuelva el adapter real y no un mock.
func newNearbyMgrWithStore(t *testing.T) (ridersPorts.RidersApp, ridersPorts.RiderPresenceStore) {
	t.Helper()
	repo := mocksriders.NewRidersRepo(t)
	stream := newRidersStream(t)
	bcv := mockbcvports.NewBcvApp(t)
	tripsRepo := mocksriders.NewTripsStore(t)
	earnings := mocksriders.NewTripEarningRecorder(t)
	repo.On("SearchActiveOrdersByRiderID", mock.Anything, mock.Anything).Return(nil, errcustom.ErrNotFound).Maybe()
	repo.On("SearchAvailableOrders", mock.Anything).Return(nil, errcustom.ErrNotFound).Maybe()

	store := newPresenceStore(t)
	return app.New(repo, stream, bcv, tripsRepo, earnings, nil, nil, nil, nil, store), store
}

// newPresenceStore arma el adapter real contra un Redis en memoria.
func newPresenceStore(t *testing.T) ridersPorts.RiderPresenceStore {
	t.Helper()
	client := redis.NewClient(&redis.Options{Addr: miniredis.RunT(t).Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return ridersRepo.NewRiderPresenceRepo(client)
}

// newRidersStream arma el mock de stream con un catch-all para
// PublisherEvent: casi ninguno de estos tests le importa qué evento se
// publicó (trip_status_changed, trip_no_driver...), solo que no explote.
// Un test que sí le importa stubea PublisherEvent explícitamente aparte.
func newRidersStream(t *testing.T) *mocksriders.RidersStream {
	t.Helper()
	stream := mocksriders.NewRidersStream(t)
	stream.On("PublisherEvent", mock.Anything, mock.Anything).Return(nil).Maybe()
	return stream
}

// ageStoredLocation reescribe la presencia con una marca vieja: la frescura del
// mapa la decide updated_at en Redis, no el mapa en memoria.
func ageStoredLocation(t *testing.T, store ridersPorts.RiderPresenceStore, riderID uuid.UUID, lat, lng float64, age time.Duration) {
	t.Helper()
	cell, err := geo.LatLngToCell(lat, lng, geo.DriverMatchResolution)
	require.NoError(t, err)
	require.NoError(t, store.SaveLocation(context.Background(), riderModels.RiderPresence{
		RiderID: riderID,
		Location: riderModels.RiderLocation{
			Latitude:  lat,
			Longitude: lng,
			H3Cell:    cell,
			UpdatedAt: time.Now().Add(-age),
		},
		Mode: riderModels.RiderModeTrips,
	}))
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

	pins, err := asNearby(t, svc).NearbyRiders(context.Background(), lat, lng)

	require.NoError(t, err)
	require.Len(t, pins, 1)
	require.Equal(t, 10.481, pins[0].Latitude)
	require.Equal(t, -66.904, pins[0].Longitude)
}

func TestNearbyRiders_OrdersModeAndBusyAndStaleAreExcluded(t *testing.T) {
	svc, store := newNearbyMgrWithStore(t)
	lat, lng := 10.4806, -66.9036

	placeRider(t, svc, lat, lng, riderModels.RiderModeOrders)

	busyID, _ := placeTripRider(t, svc, lat, lng)
	app.MarkRiderBusyForTest(svc, busyID)

	staleID, _ := placeTripRider(t, svc, lat+0.001, lng)
	ageStoredLocation(t, store, staleID, lat+0.001, lng, riderModels.NearbyRiderMaxAge+time.Second)

	pins, err := asNearby(t, svc).NearbyRiders(context.Background(), lat, lng)

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

	pins, err := asNearby(t, svc).NearbyRiders(context.Background(), lat, lng)

	require.NoError(t, err)
	require.Len(t, pins, riderModels.NearbyRiderMaxResults)
	wantLat := math.Round(lat*riderModels.NearbyCoordDecimals) / riderModels.NearbyCoordDecimals
	require.Equal(t, wantLat, pins[0].Latitude)
}

// Acota el radio por los dos lados: un rider al borde del disco entra y el del
// anillo siguiente queda afuera. Sin el caso de adentro, un radio más chico de
// lo debido pasa desapercibido.
func TestNearbyRiders_KeepsRidersAtTheEdgeOfTheDisk(t *testing.T) {
	svc := newNearbyMgr(t)
	lat, lng := 10.4806, -66.9036
	origin, err := geo.LatLngToCell(lat, lng, geo.DriverMatchResolution)
	require.NoError(t, err)

	edgeLat, edgeLng := latLngAtRing(t, origin, riderModels.NearbyRiderRings)
	placeTripRider(t, svc, edgeLat, edgeLng)
	outLat, outLng := latLngAtRing(t, origin, riderModels.NearbyRiderRings+1)
	placeTripRider(t, svc, outLat, outLng)

	pins, err := asNearby(t, svc).NearbyRiders(context.Background(), lat, lng)

	require.NoError(t, err)
	require.Len(t, pins, 1)
	require.Equal(t, roundTo3(edgeLat), pins[0].Latitude)
}

func roundTo3(v float64) float64 {
	return math.Round(v*riderModels.NearbyCoordDecimals) / riderModels.NearbyCoordDecimals
}

func TestNearbyRiders_UnregisterRemovesThePin(t *testing.T) {
	svc := newNearbyMgr(t)
	lat, lng := 10.4806, -66.9036
	id, conn := placeTripRider(t, svc, lat, lng)
	conn.On("Close").Return()
	svc.Unregister(id, conn)

	pins, err := asNearby(t, svc).NearbyRiders(context.Background(), lat, lng)
	require.NoError(t, err)
	require.Empty(t, pins)
}
