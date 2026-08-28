package app_test

import (
	"context"
	"errors"
	"testing"

	mockbcvports "Encargalo.app-api.go/internal/mocks/bcv/domain/ports"
	mocksriders "Encargalo.app-api.go/internal/mocks/riders/domain/ports"
	ordersModels "Encargalo.app-api.go/internal/orders/domain/models"
	ordersPorts "Encargalo.app-api.go/internal/orders/domain/ports"
	"Encargalo.app-api.go/internal/pkg/geo"
	"Encargalo.app-api.go/internal/riders/app"
	riderModels "Encargalo.app-api.go/internal/riders/domain/models"
	ridersPorts "Encargalo.app-api.go/internal/riders/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// asBroadcaster expone BroadcastNewOrder/BroadcastOrderUpdate, que orders/app
// consume vía ordersPorts.OrderRiderBroadcaster (ver di.go: mgr.(ordersPorts.OrderRiderBroadcaster)).
// No forman parte de ridersPorts.RidersApp, así que se necesita el type assertion.
func asBroadcaster(t *testing.T, svc ridersPorts.RidersApp) ordersPorts.OrderRiderBroadcaster {
	t.Helper()
	b, ok := svc.(ordersPorts.OrderRiderBroadcaster)
	require.True(t, ok, "appRidersManager debe implementar ordersPorts.OrderRiderBroadcaster")
	return b
}

func TestRegister_SendsActiveAndAvailableOrders(t *testing.T) {
	repo := mocksriders.NewRidersRepo(t)
	stream := mocksriders.NewRidersStream(t)
	bcv := mockbcvports.NewBcvApp(t)
	svc := app.New(repo, stream, bcv)

	riderID := uuid.New()
	conn := mocksriders.NewRiderConn(t)

	activeOrder := ordersModels.Order{ID: uuid.New(), ShopID: uuid.New(), Status: string(ordersModels.StatusOnTheWay)}
	availableOrder := ordersModels.Order{ID: uuid.New(), ShopID: uuid.New(), Status: string(ordersModels.StatusReady)}

	repo.On("SearchActiveOrdersByRiderID", mock.Anything, riderID).Return([]ordersModels.Order{activeOrder}, nil)
	repo.On("SearchAvailableOrders", mock.Anything).Return([]ordersModels.Order{availableOrder}, nil)
	repo.On("SearchShopByID", mock.Anything, mock.Anything).Return(&ordersModels.ShopOrder{}, nil)
	bcv.On("GetRate", mock.Anything).Return(nil, errors.New("bcv down"))
	conn.On("Send", mock.Anything).Return().Twice()

	err := svc.Register(context.Background(), riderID, conn, riderModels.RiderModeOrders)

	require.NoError(t, err)
}

func TestRegister_NoActiveOrNotFoundIsNotAnError(t *testing.T) {
	repo := mocksriders.NewRidersRepo(t)
	stream := mocksriders.NewRidersStream(t)
	bcv := mockbcvports.NewBcvApp(t)
	svc := app.New(repo, stream, bcv)

	riderID := uuid.New()
	conn := mocksriders.NewRiderConn(t)

	repo.On("SearchActiveOrdersByRiderID", mock.Anything, riderID).Return(nil, errcustom.ErrNotFound)
	repo.On("SearchAvailableOrders", mock.Anything).Return(nil, errcustom.ErrNotFound)

	err := svc.Register(context.Background(), riderID, conn, riderModels.RiderModeOrders)

	require.NoError(t, err)
}

func TestRegister_RepoErrorPropagates(t *testing.T) {
	repo := mocksriders.NewRidersRepo(t)
	stream := mocksriders.NewRidersStream(t)
	bcv := mockbcvports.NewBcvApp(t)
	svc := app.New(repo, stream, bcv)

	riderID := uuid.New()
	conn := mocksriders.NewRiderConn(t)

	repo.On("SearchActiveOrdersByRiderID", mock.Anything, riderID).Return(nil, errcustom.ErrUnexpectedError)

	err := svc.Register(context.Background(), riderID, conn, riderModels.RiderModeOrders)

	assert.ErrorIs(t, err, errcustom.ErrUnexpectedError)
}

func TestUnregister_ClosesConnAndRemovesOnlyIfSameConn(t *testing.T) {
	repo := mocksriders.NewRidersRepo(t)
	stream := mocksriders.NewRidersStream(t)
	bcv := mockbcvports.NewBcvApp(t)
	svc := app.New(repo, stream, bcv)

	riderID := uuid.New()
	conn := mocksriders.NewRiderConn(t)
	conn.On("Close").Return()

	svc.Unregister(riderID, conn)
}

func TestBroadcastNewOrder_SendsToAvailableRiders(t *testing.T) {
	repo := mocksriders.NewRidersRepo(t)
	stream := mocksriders.NewRidersStream(t)
	bcv := mockbcvports.NewBcvApp(t)
	svc := app.New(repo, stream, bcv)
	broadcaster := asBroadcaster(t, svc)

	riderID := uuid.New()
	conn := mocksriders.NewRiderConn(t)
	conn.On("Send", mock.Anything).Return()
	repo.On("SearchActiveOrdersByRiderID", mock.Anything, riderID).Return(nil, errcustom.ErrNotFound)
	repo.On("SearchAvailableOrders", mock.Anything).Return(nil, errcustom.ErrNotFound)
	require.NoError(t, svc.Register(context.Background(), riderID, conn, riderModels.RiderModeOrders))

	// BroadcastNewOrder no convierte a Bs (a diferencia de Register/BroadcastOrderUpdate):
	// el rider ve el monto USD hasta que acepta, por eso no hay expectativa de bcv.GetRate aquí.
	order := ordersModels.Order{ID: uuid.New(), ShopID: uuid.New(), Status: string(ordersModels.StatusReady)}
	repo.On("SearchShopByID", mock.Anything, order.ShopID).Return(&ordersModels.ShopOrder{}, nil)

	broadcaster.BroadcastNewOrder(context.Background(), order)
}

func TestBroadcastOrderUpdate_OnlySendsToAssignedRider(t *testing.T) {
	repo := mocksriders.NewRidersRepo(t)
	stream := mocksriders.NewRidersStream(t)
	bcv := mockbcvports.NewBcvApp(t)
	svc := app.New(repo, stream, bcv)
	broadcaster := asBroadcaster(t, svc)

	riderID := uuid.New()
	orderID := uuid.New()
	conn := mocksriders.NewRiderConn(t)
	conn.On("Send", mock.Anything).Return()
	repo.On("SearchActiveOrdersByRiderID", mock.Anything, riderID).Return([]ordersModels.Order{{ID: orderID, ShopID: uuid.New(), Status: string(ordersModels.StatusOnTheWay)}}, nil)
	repo.On("SearchAvailableOrders", mock.Anything).Return(nil, errcustom.ErrNotFound)
	repo.On("SearchShopByID", mock.Anything, mock.Anything).Return(&ordersModels.ShopOrder{}, nil)
	bcv.On("GetRate", mock.Anything).Return(nil, errors.New("bcv down"))
	require.NoError(t, svc.Register(context.Background(), riderID, conn, riderModels.RiderModeOrders))

	updated := ordersModels.Order{ID: orderID, ShopID: uuid.New(), Status: string(ordersModels.StatusCompleted)}
	broadcaster.BroadcastOrderUpdate(context.Background(), updated)

	other := ordersModels.Order{ID: uuid.New(), ShopID: uuid.New(), Status: string(ordersModels.StatusCompleted)}
	broadcaster.BroadcastOrderUpdate(context.Background(), other)
}

func TestAcceptOrder_Success(t *testing.T) {
	repo := mocksriders.NewRidersRepo(t)
	stream := mocksriders.NewRidersStream(t)
	bcv := mockbcvports.NewBcvApp(t)
	svc := app.New(repo, stream, bcv)

	riderID := uuid.New()
	orderID := uuid.New()
	conn := mocksriders.NewRiderConn(t)

	order := &ordersModels.Order{ID: orderID, ShopID: uuid.New(), Status: string(ordersModels.StatusReady)}
	repo.On("SearchOrdersByID", mock.Anything, orderID).Return(order, nil).Once()
	repo.On("AssignRider", mock.Anything, orderID, riderID).Return(nil)
	repo.On("SearchOrdersByID", mock.Anything, orderID).Return(order, nil).Once()
	repo.On("SearchShopByID", mock.Anything, order.ShopID).Return(&ordersModels.ShopOrder{}, nil)
	bcv.On("GetRate", mock.Anything).Return(nil, errors.New("bcv down"))
	conn.On("Send", mock.Anything).Return()

	err := svc.AcceptOrder(context.Background(), riderID, orderID, conn)

	require.NoError(t, err)
}

func TestAcceptOrder_InvalidStatus(t *testing.T) {
	repo := mocksriders.NewRidersRepo(t)
	stream := mocksriders.NewRidersStream(t)
	bcv := mockbcvports.NewBcvApp(t)
	svc := app.New(repo, stream, bcv)

	riderID, orderID := uuid.New(), uuid.New()
	repo.On("SearchOrdersByID", mock.Anything, orderID).Return(&ordersModels.Order{ID: orderID, Status: string(ordersModels.StatusPending)}, nil)

	err := svc.AcceptOrder(context.Background(), riderID, orderID, mocksriders.NewRiderConn(t))

	assert.ErrorIs(t, err, errcustom.ErrInvalidTransition)
}

func TestAcceptOrder_AlreadyAssignedNotifiesOrderTaken(t *testing.T) {
	repo := mocksriders.NewRidersRepo(t)
	stream := mocksriders.NewRidersStream(t)
	bcv := mockbcvports.NewBcvApp(t)
	svc := app.New(repo, stream, bcv)

	riderID, orderID := uuid.New(), uuid.New()
	conn := mocksriders.NewRiderConn(t)
	repo.On("SearchOrdersByID", mock.Anything, orderID).Return(&ordersModels.Order{ID: orderID, Status: string(ordersModels.StatusReady)}, nil)
	repo.On("AssignRider", mock.Anything, orderID, riderID).Return(errcustom.ErrOrderAlreadyAssigned)
	conn.On("Send", mock.Anything).Return()

	err := svc.AcceptOrder(context.Background(), riderID, orderID, conn)

	require.NoError(t, err)
}

func TestAcceptOrder_OrderNotFound(t *testing.T) {
	repo := mocksriders.NewRidersRepo(t)
	stream := mocksriders.NewRidersStream(t)
	bcv := mockbcvports.NewBcvApp(t)
	svc := app.New(repo, stream, bcv)

	orderID := uuid.New()
	repo.On("SearchOrdersByID", mock.Anything, orderID).Return(nil, errcustom.ErrNotFound)

	err := svc.AcceptOrder(context.Background(), uuid.New(), orderID, mocksriders.NewRiderConn(t))

	assert.ErrorIs(t, err, errcustom.ErrNotFound)
}

func TestUpdateLocation_SnapshotOnlyIncludesTripsMode(t *testing.T) {
	repo := mocksriders.NewRidersRepo(t)
	stream := mocksriders.NewRidersStream(t)
	bcv := mockbcvports.NewBcvApp(t)
	svc := app.New(repo, stream, bcv)

	tripsRiderID, ordersRiderID := uuid.New(), uuid.New()
	tripsConn, ordersConn := mocksriders.NewRiderConn(t), mocksriders.NewRiderConn(t)
	repo.On("SearchActiveOrdersByRiderID", mock.Anything, mock.Anything).Return(nil, errcustom.ErrNotFound)
	repo.On("SearchAvailableOrders", mock.Anything).Return(nil, errcustom.ErrNotFound)

	require.NoError(t, svc.Register(context.Background(), tripsRiderID, tripsConn, riderModels.RiderModeTrips))
	require.NoError(t, svc.Register(context.Background(), ordersRiderID, ordersConn, riderModels.RiderModeOrders))

	lat, lng := 10.4806, -66.9036
	wantCell, err := geo.LatLngToCell(lat, lng, geo.DriverMatchResolution)
	require.NoError(t, err)

	require.NoError(t, svc.UpdateLocation(context.Background(), tripsRiderID, lat, lng))
	require.NoError(t, svc.UpdateLocation(context.Background(), ordersRiderID, lat, lng))

	snapshot := svc.SnapshotTripsRiders()

	require.Len(t, snapshot, 1)
	assert.Equal(t, tripsRiderID, snapshot[0].RiderID)
	assert.Equal(t, wantCell, snapshot[0].Location.H3Cell)
}

func TestBroadcastNewOrder_SkipsRidersInTripsMode(t *testing.T) {
	repo := mocksriders.NewRidersRepo(t)
	stream := mocksriders.NewRidersStream(t)
	bcv := mockbcvports.NewBcvApp(t)
	svc := app.New(repo, stream, bcv)
	broadcaster := asBroadcaster(t, svc)

	tripsRiderID, ordersRiderID := uuid.New(), uuid.New()
	tripsConn, ordersConn := mocksriders.NewRiderConn(t), mocksriders.NewRiderConn(t)
	ordersConn.On("Send", mock.Anything).Return()
	repo.On("SearchActiveOrdersByRiderID", mock.Anything, mock.Anything).Return(nil, errcustom.ErrNotFound)
	repo.On("SearchAvailableOrders", mock.Anything).Return(nil, errcustom.ErrNotFound)

	require.NoError(t, svc.Register(context.Background(), tripsRiderID, tripsConn, riderModels.RiderModeTrips))
	require.NoError(t, svc.Register(context.Background(), ordersRiderID, ordersConn, riderModels.RiderModeOrders))

	order := ordersModels.Order{ID: uuid.New(), ShopID: uuid.New(), Status: string(ordersModels.StatusReady)}
	repo.On("SearchShopByID", mock.Anything, order.ShopID).Return(&ordersModels.ShopOrder{}, nil)

	broadcaster.BroadcastNewOrder(context.Background(), order)

	tripsConn.AssertNotCalled(t, "Send", mock.Anything)
}

func TestUnregister_RemovesModeAndLocation(t *testing.T) {
	repo := mocksriders.NewRidersRepo(t)
	stream := mocksriders.NewRidersStream(t)
	bcv := mockbcvports.NewBcvApp(t)
	svc := app.New(repo, stream, bcv)

	riderID := uuid.New()
	conn := mocksriders.NewRiderConn(t)
	conn.On("Close").Return()
	repo.On("SearchActiveOrdersByRiderID", mock.Anything, riderID).Return(nil, errcustom.ErrNotFound)
	repo.On("SearchAvailableOrders", mock.Anything).Return(nil, errcustom.ErrNotFound)

	require.NoError(t, svc.Register(context.Background(), riderID, conn, riderModels.RiderModeTrips))
	require.NoError(t, svc.UpdateLocation(context.Background(), riderID, 10.48, -66.90))
	require.Len(t, svc.SnapshotTripsRiders(), 1)

	svc.Unregister(riderID, conn)

	assert.Empty(t, svc.SnapshotTripsRiders())
}
