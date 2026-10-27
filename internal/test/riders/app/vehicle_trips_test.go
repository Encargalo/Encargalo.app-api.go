package app_test

import (
	"context"
	"testing"

	mockbcvports "Encargalo.app-api.go/internal/mocks/bcv/domain/ports"
	mocksriders "Encargalo.app-api.go/internal/mocks/riders/domain/ports"
	"Encargalo.app-api.go/internal/riders/app"
	riderModels "Encargalo.app-api.go/internal/riders/domain/models"
	ridersPorts "Encargalo.app-api.go/internal/riders/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	tripsModels "Encargalo.app-api.go/internal/trips/domain/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// newTripVehicleMgr es newVehicleMgr pero expone el TripsStore: estas pruebas
// recorren los caminos de oferta que lo consultan.
func newTripVehicleMgr(t *testing.T) (ridersPorts.RidersApp, *mocksriders.VehicleLookup, *mocksriders.TripsStore) {
	t.Helper()
	repo := mocksriders.NewRidersRepo(t)
	repo.On("SearchActiveOrdersByRiderID", mock.Anything, mock.Anything).Return(nil, errcustom.ErrNotFound).Maybe()
	repo.On("SearchAvailableOrders", mock.Anything).Return(nil, errcustom.ErrNotFound).Maybe()
	vehicles := mocksriders.NewVehicleLookup(t)
	tripsStore := mocksriders.NewTripsStore(t)
	svc := app.New(repo, newRidersStream(t), mockbcvports.NewBcvApp(t), tripsStore, mocksriders.NewTripEarningRecorder(t), nil, nil, nil, vehicles, newPresenceStore(t), newOfferTimeoutStore(t))
	return svc, vehicles, tripsStore
}

// El rider que llega tarde recibe los viajes abiertos al cambiar de celda. Ese
// camino no pasa por el matcher, así que necesita su propio filtro de vehículo.
func TestOfferOpenTripsToLateRider_SkipsBike(t *testing.T) {
	svc, vehicles, _ := newTripVehicleMgr(t)
	matcher := asMatcher(t, svc)
	lat, lng := 10.4806, -66.9036

	_, motoConn := placeVehicleRider(t, svc, vehicles, new(riderModels.VehicleTypeMoto), lat, lng)
	motoConn.On("Send", mock.Anything).Return().Once()
	matcher.OfferTrip(context.Background(), samplePickupTrip(t))

	// La bici entra con el viaje ya abierto: sin Send esperado, cualquier oferta rompe la prueba.
	bikeID := uuid.New()
	vehicles.On("GetByUserID", mock.Anything, bikeID).Return(&riderModels.RiderProfile{UserID: bikeID, VehicleType: new(riderModels.VehicleTypeBike)}, nil).Once()
	bikeConn := mocksriders.NewRiderConn(t)
	require.NoError(t, svc.Register(context.Background(), bikeID, bikeConn, riderModels.RiderModeTrips))
	require.NoError(t, svc.UpdateLocation(context.Background(), bikeID, lat+0.0001, lng))

	bikeConn.AssertNotCalled(t, "Send", mock.Anything)
}

// La moto que llega tarde sí recibe el viaje: el filtro excluye la bici, no al rider tardío.
func TestOfferOpenTripsToLateRider_OffersToMoto(t *testing.T) {
	svc, vehicles, tripsStore := newTripVehicleMgr(t)
	matcher := asMatcher(t, svc)
	lat, lng := 10.4806, -66.9036

	trip := samplePickupTrip(t)
	_, firstConn := placeVehicleRider(t, svc, vehicles, new(riderModels.VehicleTypeMoto), lat, lng)
	firstConn.On("Send", mock.Anything).Return().Once()
	matcher.OfferTrip(context.Background(), trip)

	tripsStore.On("GetTripByID", mock.Anything, trip.ID).Return(&tripsModels.Trip{ID: trip.ID, Status: tripsModels.TripStatusSearching}, nil).Once()
	lateID := uuid.New()
	vehicles.On("GetByUserID", mock.Anything, lateID).Return(&riderModels.RiderProfile{UserID: lateID, VehicleType: new(riderModels.VehicleTypeMoto)}, nil).Once()
	lateConn := mocksriders.NewRiderConn(t)
	lateConn.On("Send", mock.Anything).Return().Once()
	require.NoError(t, svc.Register(context.Background(), lateID, lateConn, riderModels.RiderModeTrips))
	require.NoError(t, svc.UpdateLocation(context.Background(), lateID, lat+0.0001, lng))

	lateConn.AssertExpectations(t)
}

// Filtrar las ofertas es UX; rechazar en AcceptTrip es la regla. Pasa cuando el
// staff le cambia el vehículo a bici y el rider reconecta con la oferta vigente.
func TestAcceptTrip_RejectsBike(t *testing.T) {
	svc, vehicles, tripsStore := newTripVehicleMgr(t)
	matcher := asMatcher(t, svc)
	lat, lng := 10.4806, -66.9036

	trip := samplePickupTrip(t)
	riderID, conn := placeVehicleRider(t, svc, vehicles, new(riderModels.VehicleTypeMoto), lat, lng)
	conn.On("Send", mock.Anything).Return().Once()
	conn.On("Close").Return()
	matcher.OfferTrip(context.Background(), trip)

	svc.Unregister(riderID, conn)
	vehicles.On("GetByUserID", mock.Anything, riderID).Return(&riderModels.RiderProfile{UserID: riderID, VehicleType: new(riderModels.VehicleTypeBike)}, nil).Once()
	reconnected := mocksriders.NewRiderConn(t)
	require.NoError(t, svc.Register(context.Background(), riderID, reconnected, riderModels.RiderModeTrips))
	require.NoError(t, svc.UpdateLocation(context.Background(), riderID, lat, lng))

	tripsStore.On("GetTripByID", mock.Anything, trip.ID).Return(&tripsModels.Trip{ID: trip.ID, Status: tripsModels.TripStatusSearching}, nil).Once()

	err := svc.AcceptTrip(context.Background(), riderID, trip.ID, reconnected)

	require.ErrorIs(t, err, errcustom.ErrVehicleCannotCarryTrip)
	tripsStore.AssertNotCalled(t, "AssignTripRider", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// "Bici" del CRM es la misma bici: el upsert de staff no normaliza el campo, y
// el índice de Redis compara exacto, así que el valor se canoniza al publicarlo.
func TestNearbyRiders_ExcludesBikeWrittenWithCapital(t *testing.T) {
	svc, vehicles, _ := newTripVehicleMgr(t)
	capitalized := " Bici "
	lat, lng := 10.4806, -66.9036

	placeVehicleRider(t, svc, vehicles, &capitalized, lat, lng)

	pins, err := asNearby(t, svc).NearbyRiders(context.Background(), lat, lng)

	require.NoError(t, err)
	require.Empty(t, pins)
}
