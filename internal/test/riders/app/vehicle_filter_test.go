package app_test

import (
	"context"
	"errors"
	"testing"

	mockbcvports "Encargalo.app-api.go/internal/mocks/bcv/domain/ports"
	mocksriders "Encargalo.app-api.go/internal/mocks/riders/domain/ports"
	"Encargalo.app-api.go/internal/riders/app"
	riderModels "Encargalo.app-api.go/internal/riders/domain/models"
	ridersPorts "Encargalo.app-api.go/internal/riders/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newVehicleMgr(t *testing.T) (ridersPorts.RidersApp, *mocksriders.VehicleLookup) {
	t.Helper()
	repo := mocksriders.NewRidersRepo(t)
	repo.On("SearchActiveOrdersByRiderID", mock.Anything, mock.Anything).Return(nil, errcustom.ErrNotFound).Maybe()
	repo.On("SearchAvailableOrders", mock.Anything).Return(nil, errcustom.ErrNotFound).Maybe()
	vehicles := mocksriders.NewVehicleLookup(t)
	svc := app.New(repo, newRidersStream(t), mockbcvports.NewBcvApp(t), mocksriders.NewTripsStore(t), mocksriders.NewTripEarningRecorder(t), nil, nil, nil, vehicles, newPresenceStore(t))
	return svc, vehicles
}

// placeVehicleRider conecta un rider en modo viajes con el vehículo indicado; nil deja el perfil sin cargar.
func placeVehicleRider(t *testing.T, svc ridersPorts.RidersApp, vehicles *mocksriders.VehicleLookup, vehicle *string, lat, lng float64) (uuid.UUID, *mocksriders.RiderConn) {
	t.Helper()
	id := uuid.New()
	vehicles.On("GetByUserID", mock.Anything, id).Return(&riderModels.RiderProfile{UserID: id, VehicleType: vehicle}, nil).Once()
	conn := mocksriders.NewRiderConn(t)
	require.NoError(t, svc.Register(context.Background(), id, conn, riderModels.RiderModeTrips))
	require.NoError(t, svc.UpdateLocation(context.Background(), id, lat, lng))
	return id, conn
}

func TestNearbyRiders_KeepsMotoAndCar(t *testing.T) {
	svc, vehicles := newVehicleMgr(t)
	lat, lng := 10.4806, -66.9036

	placeVehicleRider(t, svc, vehicles, new(riderModels.VehicleTypeMoto), lat, lng)
	placeVehicleRider(t, svc, vehicles, new(riderModels.VehicleTypeCar), lat+0.0001, lng)

	pins, err := asNearby(t, svc).NearbyRiders(context.Background(), lat, lng)

	require.NoError(t, err)
	require.Len(t, pins, 2)
}

// Si el perfil no se puede leer, el rider no se castiga: queda elegible.
func TestNearbyRiders_KeepsRiderWhenVehicleLookupFails(t *testing.T) {
	svc, vehicles := newVehicleMgr(t)

	id := uuid.New()
	vehicles.On("GetByUserID", mock.Anything, id).Return(nil, errors.New("boom")).Once()
	require.NoError(t, svc.Register(context.Background(), id, mocksriders.NewRiderConn(t), riderModels.RiderModeTrips))
	require.NoError(t, svc.UpdateLocation(context.Background(), id, 10.4806, -66.9036))

	pins, err := asNearby(t, svc).NearbyRiders(context.Background(), 10.4806, -66.9036)

	require.NoError(t, err)
	require.Len(t, pins, 1)
}
func TestUnregister_ForgetsTheVehicle(t *testing.T) {
	svc, vehicles := newVehicleMgr(t)
	lat, lng := 10.4806, -66.9036

	id := uuid.New()
	vehicles.On("GetByUserID", mock.Anything, id).Return(&riderModels.RiderProfile{UserID: id, VehicleType: new(riderModels.VehicleTypeBike)}, nil).Once()
	conn := mocksriders.NewRiderConn(t)
	conn.On("Close").Return()
	require.NoError(t, svc.Register(context.Background(), id, conn, riderModels.RiderModeTrips))
	require.NoError(t, svc.UpdateLocation(context.Background(), id, lat, lng))
	pins, err := asNearby(t, svc).NearbyRiders(context.Background(), lat, lng)
	require.NoError(t, err)
	require.Empty(t, pins)

	svc.Unregister(id, conn)

	// Reconecta con la moto que el staff le cargó entretanto: el vehículo viejo no debe quedar pegado.
	vehicles.On("GetByUserID", mock.Anything, id).Return(&riderModels.RiderProfile{UserID: id, VehicleType: new(riderModels.VehicleTypeMoto)}, nil).Once()
	reconnected := mocksriders.NewRiderConn(t)
	require.NoError(t, svc.Register(context.Background(), id, reconnected, riderModels.RiderModeTrips))
	require.NoError(t, svc.UpdateLocation(context.Background(), id, lat, lng))

	pins, err = asNearby(t, svc).NearbyRiders(context.Background(), lat, lng)
	require.NoError(t, err)
	require.Len(t, pins, 1)
}
func TestOfferTrip_BikeRiderDoesNotTakeACandidateSlot(t *testing.T) {
	svc, vehicles := newVehicleMgr(t)
	matcher := asMatcher(t, svc)

	// El más cercano es una bici: su cupo va a los tres de moto que siguen.
	_, bikeConn := placeVehicleRider(t, svc, vehicles, new(riderModels.VehicleTypeBike), 10.4808, -66.9036)
	motoConns := make([]*mocksriders.RiderConn, 0, 3)
	for _, lat := range []float64{10.4810, 10.4815, 10.4850} {
		_, conn := placeVehicleRider(t, svc, vehicles, new(riderModels.VehicleTypeMoto), lat, -66.9036)
		conn.On("Send", mock.Anything).Return().Once()
		motoConns = append(motoConns, conn)
	}

	matcher.OfferTrip(context.Background(), samplePickupTrip(t))

	for _, conn := range motoConns {
		conn.AssertExpectations(t)
	}
	bikeConn.AssertNotCalled(t, "Send", mock.Anything)
}

// El mapa del cliente ahora filtra igual que el matching: antes una bici
// aparecía como pin disponible aunque nunca pudiera aceptar un viaje.
func TestNearbyRiders_ExcludesBikeRiders(t *testing.T) {
	svc, vehicles := newVehicleMgr(t)
	lat, lng := 10.4806, -66.9036

	placeVehicleRider(t, svc, vehicles, new(riderModels.VehicleTypeMoto), lat, lng)
	placeVehicleRider(t, svc, vehicles, new(riderModels.VehicleTypeBike), lat+0.0001, lng)

	pins, err := asNearby(t, svc).NearbyRiders(context.Background(), lat, lng)

	require.NoError(t, err)
	require.Len(t, pins, 1)
}

// Mismo criterio que el matching: sin vehículo cargado el rider sigue elegible.
func TestNearbyRiders_KeepsRiderWithoutVehicleLoaded(t *testing.T) {
	svc, vehicles := newVehicleMgr(t)

	placeVehicleRider(t, svc, vehicles, nil, 10.4806, -66.9036)

	pins, err := asNearby(t, svc).NearbyRiders(context.Background(), 10.4806, -66.9036)

	require.NoError(t, err)
	require.Len(t, pins, 1)
}
