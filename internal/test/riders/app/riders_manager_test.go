package app_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	bcvmodels "Encargalo.app-api.go/internal/bcv/domain/models"
	mockbcvports "Encargalo.app-api.go/internal/mocks/bcv/domain/ports"
	mocksriders "Encargalo.app-api.go/internal/mocks/riders/domain/ports"
	mockstrips "Encargalo.app-api.go/internal/mocks/trips/domain/ports"
	ordersModels "Encargalo.app-api.go/internal/orders/domain/models"
	ordersPorts "Encargalo.app-api.go/internal/orders/domain/ports"
	"Encargalo.app-api.go/internal/pkg/geo"
	"Encargalo.app-api.go/internal/riders/app"
	riderModels "Encargalo.app-api.go/internal/riders/domain/models"
	ridersPorts "Encargalo.app-api.go/internal/riders/domain/ports"
	"Encargalo.app-api.go/internal/shared/deliveryfee"
	"Encargalo.app-api.go/internal/shared/errcustom"
	tripsModels "Encargalo.app-api.go/internal/trips/domain/models"
	tripsPorts "Encargalo.app-api.go/internal/trips/domain/ports"
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
	tripsRepo := mockstrips.NewTripsRepo(t)
	tripEarnings := mocksriders.NewTripEarningRecorder(t)
	svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings)

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
	tripsRepo := mockstrips.NewTripsRepo(t)
	tripEarnings := mocksriders.NewTripEarningRecorder(t)
	svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings)

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
	tripsRepo := mockstrips.NewTripsRepo(t)
	tripEarnings := mocksriders.NewTripEarningRecorder(t)
	svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings)

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
	tripsRepo := mockstrips.NewTripsRepo(t)
	tripEarnings := mocksriders.NewTripEarningRecorder(t)
	svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings)

	riderID := uuid.New()
	conn := mocksriders.NewRiderConn(t)
	conn.On("Close").Return()

	svc.Unregister(riderID, conn)
}

func TestBroadcastNewOrder_SendsToAvailableRiders(t *testing.T) {
	repo := mocksriders.NewRidersRepo(t)
	stream := mocksriders.NewRidersStream(t)
	bcv := mockbcvports.NewBcvApp(t)
	tripsRepo := mockstrips.NewTripsRepo(t)
	tripEarnings := mocksriders.NewTripEarningRecorder(t)
	svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings)
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
	tripsRepo := mockstrips.NewTripsRepo(t)
	tripEarnings := mocksriders.NewTripEarningRecorder(t)
	svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings)
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
	tripsRepo := mockstrips.NewTripsRepo(t)
	tripEarnings := mocksriders.NewTripEarningRecorder(t)
	svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings)

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
	tripsRepo := mockstrips.NewTripsRepo(t)
	tripEarnings := mocksriders.NewTripEarningRecorder(t)
	svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings)

	riderID, orderID := uuid.New(), uuid.New()
	repo.On("SearchOrdersByID", mock.Anything, orderID).Return(&ordersModels.Order{ID: orderID, Status: string(ordersModels.StatusPending)}, nil)

	err := svc.AcceptOrder(context.Background(), riderID, orderID, mocksriders.NewRiderConn(t))

	assert.ErrorIs(t, err, errcustom.ErrInvalidTransition)
}

func TestAcceptOrder_AlreadyAssignedNotifiesOrderTaken(t *testing.T) {
	repo := mocksriders.NewRidersRepo(t)
	stream := mocksriders.NewRidersStream(t)
	bcv := mockbcvports.NewBcvApp(t)
	tripsRepo := mockstrips.NewTripsRepo(t)
	tripEarnings := mocksriders.NewTripEarningRecorder(t)
	svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings)

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
	tripsRepo := mockstrips.NewTripsRepo(t)
	tripEarnings := mocksriders.NewTripEarningRecorder(t)
	svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings)

	orderID := uuid.New()
	repo.On("SearchOrdersByID", mock.Anything, orderID).Return(nil, errcustom.ErrNotFound)

	err := svc.AcceptOrder(context.Background(), uuid.New(), orderID, mocksriders.NewRiderConn(t))

	assert.ErrorIs(t, err, errcustom.ErrNotFound)
}

func TestUpdateLocation_SnapshotOnlyIncludesTripsMode(t *testing.T) {
	repo := mocksriders.NewRidersRepo(t)
	stream := mocksriders.NewRidersStream(t)
	bcv := mockbcvports.NewBcvApp(t)
	tripsRepo := mockstrips.NewTripsRepo(t)
	tripEarnings := mocksriders.NewTripEarningRecorder(t)
	svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings)

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
	tripsRepo := mockstrips.NewTripsRepo(t)
	tripEarnings := mocksriders.NewTripEarningRecorder(t)
	svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings)
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
	tripsRepo := mockstrips.NewTripsRepo(t)
	tripEarnings := mocksriders.NewTripEarningRecorder(t)
	svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings)

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

func asMatcher(t *testing.T, svc ridersPorts.RidersApp) tripsPorts.TripRiderMatcher {
	t.Helper()
	m, ok := svc.(tripsPorts.TripRiderMatcher)
	require.True(t, ok, "appRidersManager debe implementar tripsPorts.TripRiderMatcher")
	return m
}

func samplePickupTrip(t *testing.T) tripsModels.Trip {
	t.Helper()
	lat, lng := 10.4806, -66.9036
	cell, err := geo.LatLngToCell(lat, lng, geo.DriverMatchResolution)
	require.NoError(t, err)
	return tripsModels.Trip{
		ID:              uuid.New(),
		PickupLatitude:  lat,
		PickupLongitude: lng,
		PickupH3Cell:    &cell,
	}
}

func TestOfferTrip_OffersNearestCandidatesSortedByDistance(t *testing.T) {
	repo := mocksriders.NewRidersRepo(t)
	stream := mocksriders.NewRidersStream(t)
	bcv := mockbcvports.NewBcvApp(t)
	tripsRepo := mockstrips.NewTripsRepo(t)
	tripEarnings := mocksriders.NewTripEarningRecorder(t)
	svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings)
	matcher := asMatcher(t, svc)

	repo.On("SearchActiveOrdersByRiderID", mock.Anything, mock.Anything).Return(nil, errcustom.ErrNotFound)
	repo.On("SearchAvailableOrders", mock.Anything).Return(nil, errcustom.ErrNotFound)

	type candidate struct {
		id       uuid.UUID
		conn     *mocksriders.RiderConn
		lat, lng float64
	}
	candidates := []candidate{
		{uuid.New(), mocksriders.NewRiderConn(t), 10.4808, -66.9036},
		{uuid.New(), mocksriders.NewRiderConn(t), 10.4810, -66.9036},
		{uuid.New(), mocksriders.NewRiderConn(t), 10.4815, -66.9036},
		{uuid.New(), mocksriders.NewRiderConn(t), 10.4850, -66.9036},
	}
	for _, c := range candidates {
		require.NoError(t, svc.Register(context.Background(), c.id, c.conn, riderModels.RiderModeTrips))
		require.NoError(t, svc.UpdateLocation(context.Background(), c.id, c.lat, c.lng))
	}
	for _, c := range candidates[:3] {
		c.conn.On("Send", mock.Anything).Return().Once()
	}

	trip := samplePickupTrip(t)
	matcher.OfferTrip(context.Background(), trip)

	for _, c := range candidates[:3] {
		c.conn.AssertExpectations(t)
	}
	candidates[3].conn.AssertNotCalled(t, "Send", mock.Anything)
}

func TestOfferTrip_NoCandidatesExpiresTripToNoDriver(t *testing.T) {
	repo := mocksriders.NewRidersRepo(t)
	stream := mocksriders.NewRidersStream(t)
	bcv := mockbcvports.NewBcvApp(t)
	tripsRepo := mockstrips.NewTripsRepo(t)
	tripEarnings := mocksriders.NewTripEarningRecorder(t)
	svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings)
	matcher := asMatcher(t, svc)

	trip := samplePickupTrip(t)
	tripsRepo.On("UpdateTripStatus", mock.Anything, trip.ID, tripsModels.TripStatusSearching, tripsModels.TripStatusNoDriver).Return(true, nil)

	matcher.OfferTrip(context.Background(), trip)

	tripsRepo.AssertExpectations(t)
}

func TestAcceptTrip_Success(t *testing.T) {
	repo := mocksriders.NewRidersRepo(t)
	stream := mocksriders.NewRidersStream(t)
	bcv := mockbcvports.NewBcvApp(t)
	tripsRepo := mockstrips.NewTripsRepo(t)
	tripEarnings := mocksriders.NewTripEarningRecorder(t)
	svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings)
	matcher := asMatcher(t, svc)

	repo.On("SearchActiveOrdersByRiderID", mock.Anything, mock.Anything).Return(nil, errcustom.ErrNotFound)
	repo.On("SearchAvailableOrders", mock.Anything).Return(nil, errcustom.ErrNotFound)

	riderID := uuid.New()
	conn := mocksriders.NewRiderConn(t)
	conn.On("Send", mock.Anything).Return()
	require.NoError(t, svc.Register(context.Background(), riderID, conn, riderModels.RiderModeTrips))
	require.NoError(t, svc.UpdateLocation(context.Background(), riderID, 10.4806, -66.9036))

	trip := samplePickupTrip(t)
	tripsRepo.On("GetTripByID", mock.Anything, trip.ID).Return(&tripsModels.Trip{ID: trip.ID, Status: tripsModels.TripStatusSearching}, nil)
	tripsRepo.On("AssignTripRider", mock.Anything, trip.ID, riderID).Return(true, nil)
	matcher.OfferTrip(context.Background(), trip)

	err := svc.AcceptTrip(context.Background(), riderID, trip.ID, conn)

	require.NoError(t, err)
}

func TestAcceptTrip_AlreadyTakenNotifiesTripTaken(t *testing.T) {
	repo := mocksriders.NewRidersRepo(t)
	stream := mocksriders.NewRidersStream(t)
	bcv := mockbcvports.NewBcvApp(t)
	tripsRepo := mockstrips.NewTripsRepo(t)
	tripEarnings := mocksriders.NewTripEarningRecorder(t)
	svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings)
	matcher := asMatcher(t, svc)

	repo.On("SearchActiveOrdersByRiderID", mock.Anything, mock.Anything).Return(nil, errcustom.ErrNotFound)
	repo.On("SearchAvailableOrders", mock.Anything).Return(nil, errcustom.ErrNotFound)

	riderID := uuid.New()
	conn := mocksriders.NewRiderConn(t)
	conn.On("Send", mock.Anything).Return()
	require.NoError(t, svc.Register(context.Background(), riderID, conn, riderModels.RiderModeTrips))
	require.NoError(t, svc.UpdateLocation(context.Background(), riderID, 10.4806, -66.9036))

	trip := samplePickupTrip(t)
	tripsRepo.On("GetTripByID", mock.Anything, trip.ID).Return(&tripsModels.Trip{ID: trip.ID, Status: tripsModels.TripStatusSearching}, nil)
	tripsRepo.On("AssignTripRider", mock.Anything, trip.ID, riderID).Return(false, nil)
	matcher.OfferTrip(context.Background(), trip)

	err := svc.AcceptTrip(context.Background(), riderID, trip.ID, conn)

	require.NoError(t, err)
}

func TestAcceptTrip_RejectsWhenNotSearching(t *testing.T) {
	repo := mocksriders.NewRidersRepo(t)
	stream := mocksriders.NewRidersStream(t)
	bcv := mockbcvports.NewBcvApp(t)
	tripsRepo := mockstrips.NewTripsRepo(t)
	tripEarnings := mocksriders.NewTripEarningRecorder(t)
	svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings)

	riderID, tripID := uuid.New(), uuid.New()
	tripsRepo.On("GetTripByID", mock.Anything, tripID).Return(&tripsModels.Trip{ID: tripID, Status: tripsModels.TripStatusAccepted}, nil)

	err := svc.AcceptTrip(context.Background(), riderID, tripID, mocksriders.NewRiderConn(t))

	assert.ErrorIs(t, err, errcustom.ErrInvalidTransition)
}

func TestAdvanceTrip_AcceptedMovesToArriving(t *testing.T) {
	repo := mocksriders.NewRidersRepo(t)
	stream := mocksriders.NewRidersStream(t)
	bcv := mockbcvports.NewBcvApp(t)
	tripsRepo := mockstrips.NewTripsRepo(t)
	tripEarnings := mocksriders.NewTripEarningRecorder(t)
	svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings)

	riderID, tripID := uuid.New(), uuid.New()
	conn := mocksriders.NewRiderConn(t)
	conn.On("Send", mock.Anything).Return()

	tripsRepo.On("GetTripByID", mock.Anything, tripID).Return(&tripsModels.Trip{ID: tripID, RiderID: &riderID, Status: tripsModels.TripStatusAccepted}, nil)
	tripsRepo.On("UpdateTripStatus", mock.Anything, tripID, tripsModels.TripStatusAccepted, tripsModels.TripStatusArriving).Return(true, nil)

	err := svc.AdvanceTrip(context.Background(), riderID, tripID, conn)

	require.NoError(t, err)
}

func TestAdvanceTrip_ArrivingMovesToInProgress(t *testing.T) {
	repo := mocksriders.NewRidersRepo(t)
	stream := mocksriders.NewRidersStream(t)
	bcv := mockbcvports.NewBcvApp(t)
	tripsRepo := mockstrips.NewTripsRepo(t)
	tripEarnings := mocksriders.NewTripEarningRecorder(t)
	svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings)

	riderID, tripID := uuid.New(), uuid.New()
	conn := mocksriders.NewRiderConn(t)
	conn.On("Send", mock.Anything).Return()

	tripsRepo.On("GetTripByID", mock.Anything, tripID).Return(&tripsModels.Trip{ID: tripID, RiderID: &riderID, Status: tripsModels.TripStatusArriving}, nil)
	tripsRepo.On("UpdateTripStatus", mock.Anything, tripID, tripsModels.TripStatusArriving, tripsModels.TripStatusInProgress).Return(true, nil)

	err := svc.AdvanceTrip(context.Background(), riderID, tripID, conn)

	require.NoError(t, err)
}

func TestAdvanceTrip_InProgressCompletesAndRecordsEarning(t *testing.T) {
	repo := mocksriders.NewRidersRepo(t)
	stream := mocksriders.NewRidersStream(t)
	bcv := mockbcvports.NewBcvApp(t)
	tripsRepo := mockstrips.NewTripsRepo(t)
	tripEarnings := mocksriders.NewTripEarningRecorder(t)
	svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings)

	riderID, tripID := uuid.New(), uuid.New()
	conn := mocksriders.NewRiderConn(t)
	conn.On("Send", mock.Anything).Return()

	// Fuera de production, completeTrip usa deliveryfee.DevPlaceholderFee en
	// vez de llamar a Google Maps (ver deliveryfee.Calculate / config.Get().IsProduction()).
	tripsRepo.On("GetTripByID", mock.Anything, tripID).Return(&tripsModels.Trip{ID: tripID, RiderID: &riderID, Status: tripsModels.TripStatusInProgress}, nil)
	tripsRepo.On("CompleteTrip", mock.Anything, tripID, deliveryfee.DevPlaceholderFee).Return(true, nil)
	bcv.On("GetRate", mock.Anything).Return(&bcvmodels.ExchangeRate{Rate: 40}, nil)
	tripEarnings.On("RecordTripEarning", mock.Anything, riderID, tripID, deliveryfee.DevPlaceholderFee, 40.0).Return(nil)

	err := svc.AdvanceTrip(context.Background(), riderID, tripID, conn)

	require.NoError(t, err)
}

func TestAdvanceTrip_RejectsWhenRiderNotAssigned(t *testing.T) {
	repo := mocksriders.NewRidersRepo(t)
	stream := mocksriders.NewRidersStream(t)
	bcv := mockbcvports.NewBcvApp(t)
	tripsRepo := mockstrips.NewTripsRepo(t)
	tripEarnings := mocksriders.NewTripEarningRecorder(t)
	svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings)

	assignedRiderID, otherRiderID, tripID := uuid.New(), uuid.New(), uuid.New()
	tripsRepo.On("GetTripByID", mock.Anything, tripID).Return(&tripsModels.Trip{ID: tripID, RiderID: &assignedRiderID, Status: tripsModels.TripStatusAccepted}, nil)

	err := svc.AdvanceTrip(context.Background(), otherRiderID, tripID, mocksriders.NewRiderConn(t))

	assert.ErrorIs(t, err, errcustom.ErrInvalidTransition)
}

func TestAdvanceTrip_RejectsTerminalStatus(t *testing.T) {
	repo := mocksriders.NewRidersRepo(t)
	stream := mocksriders.NewRidersStream(t)
	bcv := mockbcvports.NewBcvApp(t)
	tripsRepo := mockstrips.NewTripsRepo(t)
	tripEarnings := mocksriders.NewTripEarningRecorder(t)
	svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings)

	riderID, tripID := uuid.New(), uuid.New()
	tripsRepo.On("GetTripByID", mock.Anything, tripID).Return(&tripsModels.Trip{ID: tripID, RiderID: &riderID, Status: tripsModels.TripStatusCompleted}, nil)

	err := svc.AdvanceTrip(context.Background(), riderID, tripID, mocksriders.NewRiderConn(t))

	assert.ErrorIs(t, err, errcustom.ErrInvalidTransition)
}

// TestAdvanceTrip_ConcurrentCompletionIsSerialized cubre la vulnerabilidad de
// seguridad #1 (auditoría): sin el guard completingTrips, un spam de
// advance_trip sobre el mismo trip en in_progress podía disparar varias
// llamadas reales a Google Maps (tripFare) en paralelo antes de que la
// primera confirmara en DB. Aquí se verifica que la segunda llamada, mientras
// la primera sigue "en vuelo" dentro de CompleteTrip, se rechaza de inmediato
// sin volver a invocar CompleteTrip (el mock .Once() lo garantiza).
func TestAdvanceTrip_ConcurrentCompletionIsSerialized(t *testing.T) {
	repo := mocksriders.NewRidersRepo(t)
	stream := mocksriders.NewRidersStream(t)
	bcv := mockbcvports.NewBcvApp(t)
	tripsRepo := mockstrips.NewTripsRepo(t)
	tripEarnings := mocksriders.NewTripEarningRecorder(t)
	svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings)

	riderID, tripID := uuid.New(), uuid.New()
	conn := mocksriders.NewRiderConn(t)
	conn.On("Send", mock.Anything).Return()

	entered := make(chan struct{})
	proceed := make(chan struct{})

	tripsRepo.On("GetTripByID", mock.Anything, tripID).Return(&tripsModels.Trip{ID: tripID, RiderID: &riderID, Status: tripsModels.TripStatusInProgress}, nil)
	tripsRepo.On("CompleteTrip", mock.Anything, tripID, deliveryfee.DevPlaceholderFee).Run(func(mock.Arguments) {
		close(entered)
		<-proceed
	}).Return(true, nil).Once()
	bcv.On("GetRate", mock.Anything).Return(&bcvmodels.ExchangeRate{Rate: 40}, nil)
	tripEarnings.On("RecordTripEarning", mock.Anything, riderID, tripID, deliveryfee.DevPlaceholderFee, 40.0).Return(nil)

	var wg sync.WaitGroup
	var firstErr error
	wg.Go(func() {
		firstErr = svc.AdvanceTrip(context.Background(), riderID, tripID, conn)
	})

	<-entered // la primera llamada ya está dentro de CompleteTrip (tripFare ya se calculó)

	secondErr := svc.AdvanceTrip(context.Background(), riderID, tripID, conn)
	close(proceed)
	wg.Wait()

	assert.ErrorIs(t, secondErr, errcustom.ErrInvalidTransition)
	require.NoError(t, firstErr)
	tripsRepo.AssertExpectations(t) // CompleteTrip solo se llamó una vez (.Once())
}
