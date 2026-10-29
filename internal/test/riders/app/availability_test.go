package app_test

import (
	"context"
	"testing"

	mockbcvports "Encargalo.app-api.go/internal/mocks/bcv/domain/ports"
	mocksriders "Encargalo.app-api.go/internal/mocks/riders/domain/ports"
	ordersModels "Encargalo.app-api.go/internal/orders/domain/models"
	"Encargalo.app-api.go/internal/riders/app"
	riderModels "Encargalo.app-api.go/internal/riders/domain/models"
	"Encargalo.app-api.go/internal/shared/errcustom"
	shopsmodels "Encargalo.app-api.go/internal/shops/domain/models/shops"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// --- Órdenes: BroadcastNewOrder respeta la pausa ---

func TestSetAvailability_PausedRiderSkipsBroadcastNewOrder(t *testing.T) {
	svc, repo, vehicles := ordersMgr(t)
	riderID, sent := connectOrdersRider(t, svc, repo, vehicles, new(riderModels.VehicleTypeMoto))
	require.NoError(t, svc.SetAvailability(context.Background(), riderID, false))

	order := orderForShop(uuid.New())
	repo.On("SearchShopByID", mock.Anything, order.ShopID).Return(&ordersModels.ShopOrder{ID: order.ShopID, Type: shopsmodels.ShopTypeRestaurant}, nil).Once()
	repo.On("SearchOrdersByID", mock.Anything, order.ID).Return(&order, nil).Maybe()

	asBroadcaster(t, svc).BroadcastNewOrder(context.Background(), order)

	require.Empty(t, sent, "un rider pausado no debe recibir new_order")
}

func TestSetAvailability_ResumeReceivesOrdersAgain(t *testing.T) {
	svc, repo, vehicles := ordersMgr(t)
	riderID, sent := connectOrdersRider(t, svc, repo, vehicles, new(riderModels.VehicleTypeMoto))
	require.NoError(t, svc.SetAvailability(context.Background(), riderID, false))
	require.NoError(t, svc.SetAvailability(context.Background(), riderID, true))

	order := orderForShop(uuid.New())
	repo.On("SearchShopByID", mock.Anything, order.ShopID).Return(&ordersModels.ShopOrder{ID: order.ShopID, Type: shopsmodels.ShopTypeRestaurant}, nil).Once()
	repo.On("SearchOrdersByID", mock.Anything, order.ID).Return(&order, nil).Maybe()

	asBroadcaster(t, svc).BroadcastNewOrder(context.Background(), order)

	require.Equal(t, order.ID.String(), orderIDOf(t, <-sent))
}

// --- Viajes: la pausa cuenta como "ocupado" para el matching ---

func TestSetAvailability_PausedRiderExcludedFromTripOffers(t *testing.T) {
	svc, tripsRepo := newBusyRiderSvc(t)
	riderID, offers := registerWatchedRider(t, svc, riderModels.RiderModeTrips)
	require.NoError(t, svc.UpdateLocation(context.Background(), riderID, 10.4806, -66.9036))
	require.NoError(t, svc.SetAvailability(context.Background(), riderID, false))

	trip := openSearchingTrip(t, svc, tripsRepo)
	asMatcher(t, svc).OfferTrip(context.Background(), trip)

	require.Equal(t, 0, offers(), "un rider pausado no debe recibir trip_offer")
}

func TestSetAvailability_ResumeReceivesTripOffersAgain(t *testing.T) {
	svc, tripsRepo := newBusyRiderSvc(t)
	riderID, offers := registerWatchedRider(t, svc, riderModels.RiderModeTrips)
	require.NoError(t, svc.UpdateLocation(context.Background(), riderID, 10.4806, -66.9036))
	require.NoError(t, svc.SetAvailability(context.Background(), riderID, false))
	require.NoError(t, svc.SetAvailability(context.Background(), riderID, true))

	trip := openSearchingTrip(t, svc, tripsRepo)
	asMatcher(t, svc).OfferTrip(context.Background(), trip)

	require.Equal(t, 1, offers())
}

// --- La pausa sobrevive a una reconexión: Register la relee desde Redis ---

func TestSetAvailability_PauseSurvivesReconnect(t *testing.T) {
	svc, repo, vehicles := ordersMgr(t)
	id := uuid.New()
	vehicles.On("GetByUserID", mock.Anything, id).Return(&riderModels.RiderProfile{UserID: id, VehicleType: new(riderModels.VehicleTypeMoto)}, nil).Maybe()
	repo.On("SearchActiveOrdersByRiderID", mock.Anything, id).Return(nil, errcustom.ErrNotFound).Maybe()
	repo.On("SearchAvailableOrders", mock.Anything).Return(nil, errcustom.ErrNotFound).Maybe()

	conn := mocksriders.NewRiderConn(t)
	conn.On("Send", mock.Anything).Return().Maybe()
	require.NoError(t, svc.Register(context.Background(), id, conn, riderModels.RiderModeOrders))
	require.NoError(t, svc.SetAvailability(context.Background(), id, false))

	// Simula la caída de red: el socket viejo se cierra y el rider reconecta
	// con uno nuevo, sin haber pedido reanudar.
	conn.On("Close").Return()
	svc.Unregister(id, conn)

	fresh, sent := newWatchedConn(t)
	require.NoError(t, svc.Register(context.Background(), id, fresh, riderModels.RiderModeOrders))

	order := orderForShop(uuid.New())
	repo.On("SearchShopByID", mock.Anything, order.ShopID).Return(&ordersModels.ShopOrder{ID: order.ShopID, Type: shopsmodels.ShopTypeRestaurant}, nil).Once()
	repo.On("SearchOrdersByID", mock.Anything, order.ID).Return(&order, nil).Maybe()

	asBroadcaster(t, svc).BroadcastNewOrder(context.Background(), order)

	require.Empty(t, sent, "la pausa debió restaurarse en memoria al reconectar")
}

func newWatchedConn(t *testing.T) (*mocksriders.RiderConn, chan []byte) {
	t.Helper()
	sent := make(chan []byte, 8)
	conn := mocksriders.NewRiderConn(t)
	conn.On("Send", mock.Anything).Run(func(args mock.Arguments) {
		sent <- args.Get(0).([]byte)
	}).Return().Maybe()
	return conn, sent
}

// --- SetAvailability sin RiderPresenceStore configurado ---

func TestSetAvailability_NoPresenceStoreFails(t *testing.T) {
	repo := mocksriders.NewRidersRepo(t)
	svc := app.New(repo, newRidersStream(t), mockbcvports.NewBcvApp(t), mocksriders.NewTripsStore(t),
		mocksriders.NewTripEarningRecorder(t), nil, nil, nil, nil, nil, newOfferTimeoutStore(t), nil, nil)

	err := svc.SetAvailability(context.Background(), uuid.New(), false)

	require.ErrorIs(t, err, errcustom.ErrUnexpectedError)
}
