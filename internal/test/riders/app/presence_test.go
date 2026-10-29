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
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newPresenceMgr(t *testing.T) (ridersPorts.RidersApp, *mocksriders.RiderPresenceStore, *mocksriders.VehicleLookup) {
	t.Helper()
	repo := mocksriders.NewRidersRepo(t)
	repo.On("SearchActiveOrdersByRiderID", mock.Anything, mock.Anything).Return(nil, errcustom.ErrNotFound).Maybe()
	repo.On("SearchAvailableOrders", mock.Anything).Return(nil, errcustom.ErrNotFound).Maybe()
	vehicles := mocksriders.NewVehicleLookup(t)
	presence := mocksriders.NewRiderPresenceStore(t)
	presence.On("IsPaused", mock.Anything, mock.Anything).Return(false, nil).Maybe()
	svc := app.New(repo, newRidersStream(t), mockbcvports.NewBcvApp(t), mocksriders.NewTripsStore(t),
		mocksriders.NewTripEarningRecorder(t), nil, nil, nil, vehicles, presence, newOfferTimeoutStore(t), nil, nil)
	return svc, presence, vehicles
}

func connectWithVehicle(t *testing.T, svc ridersPorts.RidersApp, vehicles *mocksriders.VehicleLookup, vehicle string, mode riderModels.RiderMode) (uuid.UUID, *mocksriders.RiderConn) {
	t.Helper()
	id := uuid.New()
	vehicles.On("GetByUserID", mock.Anything, id).Return(&riderModels.RiderProfile{UserID: id, VehicleType: &vehicle}, nil).Maybe()
	conn := mocksriders.NewRiderConn(t)
	require.NoError(t, svc.Register(context.Background(), id, conn, mode))
	return id, conn
}

func TestPresence_UpdateLocationPublishesLocationModeAndVehicle(t *testing.T) {
	svc, presence, vehicles := newPresenceMgr(t)
	id, _ := connectWithVehicle(t, svc, vehicles, "moto", riderModels.RiderModeTrips)

	var got riderModels.RiderPresence
	presence.On("SaveLocation", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) { got = args.Get(1).(riderModels.RiderPresence) }).
		Return(nil).Once()

	require.NoError(t, svc.UpdateLocation(context.Background(), id, 10.4806, -66.9036))

	require.Equal(t, id, got.RiderID)
	require.Equal(t, riderModels.RiderModeTrips, got.Mode)
	require.Equal(t, "moto", got.Vehicle)
	require.Equal(t, 10.4806, got.Location.Latitude)
	require.Equal(t, -66.9036, got.Location.Longitude)
	require.NotEmpty(t, got.Location.H3Cell)
	require.False(t, got.Location.UpdatedAt.IsZero())
}

// La memoria sigue siendo la fuente de verdad: si Redis falla, el ping del rider
// no puede devolver error ni cortarle el socket.
func TestPresence_SaveFailureDoesNotBreakUpdateLocation(t *testing.T) {
	svc, presence, vehicles := newPresenceMgr(t)
	id, _ := connectWithVehicle(t, svc, vehicles, "moto", riderModels.RiderModeTrips)

	presence.On("SaveLocation", mock.Anything, mock.Anything).Return(errcustom.ErrUnexpectedError).Once()

	require.NoError(t, svc.UpdateLocation(context.Background(), id, 10.4806, -66.9036))
}

func TestPresence_UnregisterRemovesPresence(t *testing.T) {
	svc, presence, vehicles := newPresenceMgr(t)
	id, conn := connectWithVehicle(t, svc, vehicles, "moto", riderModels.RiderModeTrips)
	presence.On("SaveLocation", mock.Anything, mock.Anything).Return(nil).Once()
	require.NoError(t, svc.UpdateLocation(context.Background(), id, 10.4806, -66.9036))

	presence.On("Remove", mock.Anything, id).Return(nil).Once()
	conn.On("Close").Return()

	svc.Unregister(id, conn)

	presence.AssertCalled(t, "Remove", mock.Anything, id)
}

// Unregister de una conexión que ya no es la vigente no borra la presencia del
// rider, que sigue conectado por el socket nuevo.
func TestPresence_UnregisterOfStaleConnKeepsPresence(t *testing.T) {
	svc, presence, vehicles := newPresenceMgr(t)
	id, stale := connectWithVehicle(t, svc, vehicles, "moto", riderModels.RiderModeTrips)
	presence.On("SaveLocation", mock.Anything, mock.Anything).Return(nil).Maybe()
	require.NoError(t, svc.UpdateLocation(context.Background(), id, 10.4806, -66.9036))

	fresh := mocksriders.NewRiderConn(t)
	require.NoError(t, svc.Register(context.Background(), id, fresh, riderModels.RiderModeTrips))
	stale.On("Close").Return()

	svc.Unregister(id, stale)

	presence.AssertNotCalled(t, "Remove", mock.Anything, id)
}

// Un rider que se reconecta en otro modo tiene que quedar republicado en el
// índice nuevo sin esperar su próximo ping de ubicación.
func TestPresence_ReconnectInAnotherModeRepublishesKnownLocation(t *testing.T) {
	svc, presence, vehicles := newPresenceMgr(t)
	id, _ := connectWithVehicle(t, svc, vehicles, "moto", riderModels.RiderModeTrips)
	presence.On("SaveLocation", mock.Anything, mock.Anything).Return(nil).Once()
	require.NoError(t, svc.UpdateLocation(context.Background(), id, 10.4806, -66.9036))

	var got riderModels.RiderPresence
	presence.On("SaveLocation", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) { got = args.Get(1).(riderModels.RiderPresence) }).
		Return(nil).Once()

	conn := mocksriders.NewRiderConn(t)
	require.NoError(t, svc.Register(context.Background(), id, conn, riderModels.RiderModeOrders))

	require.Equal(t, riderModels.RiderModeOrders, got.Mode)
	require.Equal(t, 10.4806, got.Location.Latitude)
}
