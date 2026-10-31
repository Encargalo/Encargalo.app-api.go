package app_test

import (
	"context"
	"errors"
	"testing"
	"time"

	mockbcvports "Encargalo.app-api.go/internal/mocks/bcv/domain/ports"
	mocksriders "Encargalo.app-api.go/internal/mocks/riders/domain/ports"
	ordersModels "Encargalo.app-api.go/internal/orders/domain/models"
	"Encargalo.app-api.go/internal/riders/app"
	riderModels "Encargalo.app-api.go/internal/riders/domain/models"
	ridersPorts "Encargalo.app-api.go/internal/riders/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// newReportLocationSvc arma el manager con un VehicleLookup mockeado y una
// presencia real (Redis en memoria), para verificar que ReportLocation —el
// camino sin socket vivo, usado por la app en background— publica lo mismo
// que update_location publicaría por WebSocket.
func newReportLocationSvc(t *testing.T, vehicleType string) (ridersPorts.RidersApp, *mocksriders.RidersRepo, ridersPorts.RiderPresenceStore) {
	t.Helper()
	repo := mocksriders.NewRidersRepo(t)
	bcv := mockbcvports.NewBcvApp(t)
	bcv.On("GetRate", mock.Anything).Return(nil, errors.New("bcv down")).Maybe()
	repo.On("SearchShopByID", mock.Anything, mock.Anything).Return(&ordersModels.ShopOrder{}, nil).Maybe()
	vehicles := mocksriders.NewVehicleLookup(t)
	vehicles.On("GetByUserID", mock.Anything, mock.Anything).Return(&riderModels.RiderProfile{VehicleType: &vehicleType}, nil).Maybe()
	store := newPresenceStore(t)
	svc := app.New(repo, newRidersStream(t), bcv, mocksriders.NewTripsStore(t), mocksriders.NewTripEarningRecorder(t), nil, nil, nil, vehicles, store, newOfferTimeoutStore(t), nil, nil, nil, nil, nil, nil)
	return svc, repo, store
}

func TestReportLocation_PublishesPresenceWithRequestedMode(t *testing.T) {
	svc, _, store := newReportLocationSvc(t, riderModels.VehicleTypeMoto)
	riderID := uuid.New()

	require.NoError(t, svc.ReportLocation(context.Background(), riderID, 10.48, -66.90, riderModels.RiderModeTrips))

	entries, err := store.SearchNearby(context.Background(), riderModels.PresenceQuery{
		Latitude: 10.48, Longitude: -66.90, RadiusKm: 1,
		Mode: riderModels.RiderModeTrips, MaxAge: time.Minute,
	})
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Equal(t, riderID, entries[0].RiderID)
}

func TestReportLocation_DoesNotLeakIntoOtherMode(t *testing.T) {
	svc, _, store := newReportLocationSvc(t, riderModels.VehicleTypeMoto)
	riderID := uuid.New()

	require.NoError(t, svc.ReportLocation(context.Background(), riderID, 10.48, -66.90, riderModels.RiderModeOrders))

	entries, err := store.SearchNearby(context.Background(), riderModels.PresenceQuery{
		Latitude: 10.48, Longitude: -66.90, RadiusKm: 1,
		Mode: riderModels.RiderModeTrips, MaxAge: time.Minute,
	})
	require.NoError(t, err)
	require.Empty(t, entries)
}

func TestReportLocation_InvalidCoordinates(t *testing.T) {
	svc, _, _ := newReportLocationSvc(t, riderModels.VehicleTypeMoto)

	err := svc.ReportLocation(context.Background(), uuid.New(), 48.85, 2.35, riderModels.RiderModeTrips)

	require.ErrorIs(t, err, errcustom.ErrInvalidCoordinates)
}

func TestReportLocation_NotifiesAssignedOrderCustomer(t *testing.T) {
	svc, repo, _ := newReportLocationSvc(t, riderModels.VehicleTypeMoto)
	mockNotifier, notifier := newRecordingOrderNotifier(t)
	setter, ok := svc.(ridersPorts.OrderCustomerNotifierSetter)
	require.True(t, ok)
	setter.SetOrderCustomerNotifier(mockNotifier)

	riderID, customerID := uuid.New(), uuid.New()
	orderID := acceptOrderFor(t, svc, repo, riderID, customerID)

	require.NoError(t, svc.ReportLocation(context.Background(), riderID, 10.48, -66.90, riderModels.RiderModeOrders))

	require.Equal(t, []orderLocationEvent{{customerID, orderID, 10.48, -66.90}}, notifier.take())
}
