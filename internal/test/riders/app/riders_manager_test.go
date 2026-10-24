package app_test

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

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
	svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings, nil, nil, nil)

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
	svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings, nil, nil, nil)

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
	svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings, nil, nil, nil)

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
	svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings, nil, nil, nil)

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
	svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings, nil, nil, nil)
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
	svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings, nil, nil, nil)
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
	svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings, nil, nil, nil)

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
	svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings, nil, nil, nil)

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
	svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings, nil, nil, nil)

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
	svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings, nil, nil, nil)

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
	svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings, nil, nil, nil)

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

func TestSnapshotTripsRiders_ExcludesStaleLocations(t *testing.T) {
	svc := newNearbyMgr(t)
	lat, lng := 10.4806, -66.9036

	freshID, _ := placeTripRider(t, svc, lat, lng)
	staleID, _ := placeTripRider(t, svc, lat+0.001, lng)
	app.AgeRiderLocationForTest(svc, staleID, riderModels.TripMatchRiderMaxAge+time.Second)

	snapshot := svc.SnapshotTripsRiders()

	require.Len(t, snapshot, 1)
	assert.Equal(t, freshID, snapshot[0].RiderID)
}

func TestSnapshotTripsRiders_KeepsLocationsTooStaleForTheMap(t *testing.T) {
	svc := newNearbyMgr(t)
	riderID, _ := placeTripRider(t, svc, 10.4806, -66.9036)
	app.AgeRiderLocationForTest(svc, riderID, riderModels.NearbyRiderMaxAge+time.Second)

	require.Len(t, svc.SnapshotTripsRiders(), 1)
}

func TestBroadcastNewOrder_SkipsRidersInTripsMode(t *testing.T) {
	repo := mocksriders.NewRidersRepo(t)
	stream := mocksriders.NewRidersStream(t)
	bcv := mockbcvports.NewBcvApp(t)
	tripsRepo := mockstrips.NewTripsRepo(t)
	tripEarnings := mocksriders.NewTripEarningRecorder(t)
	svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings, nil, nil, nil)
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
	svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings, nil, nil, nil)

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
	svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings, nil, nil, nil)
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

func TestOfferTrip_StaleRiderDoesNotTakeACandidateSlot(t *testing.T) {
	repo := mocksriders.NewRidersRepo(t)
	stream := mocksriders.NewRidersStream(t)
	bcv := mockbcvports.NewBcvApp(t)
	tripsRepo := mockstrips.NewTripsRepo(t)
	tripEarnings := mocksriders.NewTripEarningRecorder(t)
	svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings, nil, nil, nil)
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

	// El más cercano dejó de reportar: su cupo va al cuarto, que sí está reportando.
	app.AgeRiderLocationForTest(svc, candidates[0].id, riderModels.TripMatchRiderMaxAge+time.Second)
	for _, c := range candidates[1:] {
		c.conn.On("Send", mock.Anything).Return().Once()
	}

	matcher.OfferTrip(context.Background(), samplePickupTrip(t))

	for _, c := range candidates[1:] {
		c.conn.AssertExpectations(t)
	}
	candidates[0].conn.AssertNotCalled(t, "Send", mock.Anything)
}

func TestOfferTrip_NoCandidatesWaitsFullRoundsBeforeNoDriver(t *testing.T) {
	repo := mocksriders.NewRidersRepo(t)
	stream := mocksriders.NewRidersStream(t)
	bcv := mockbcvports.NewBcvApp(t)
	tripsRepo := mockstrips.NewTripsRepo(t)
	tripEarnings := mocksriders.NewTripEarningRecorder(t)
	svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings, nil, nil, nil)
	matcher := asMatcher(t, svc)

	trip := samplePickupTrip(t)
	t.Cleanup(func() { app.StopTripOfferTimerForTest(svc, trip.ID) })
	searching := trip
	searching.Status = tripsModels.TripStatusSearching
	tripsRepo.On("GetTripByID", mock.Anything, trip.ID).Return(&searching, nil)
	tripsRepo.On("UpdateTripStatus", mock.Anything, trip.ID, tripsModels.TripStatusSearching, tripsModels.TripStatusNoDriver).Return(true, nil).Once()
	tripsRepo.On("SaveTripCancelReason", mock.Anything, trip.ID, mock.Anything).Return(nil).Maybe()

	matcher.OfferTrip(context.Background(), trip)
	tripsRepo.AssertNotCalled(t, "UpdateTripStatus", mock.Anything, mock.Anything, mock.Anything, mock.Anything)

	// Rondas 2, 3 y 4 también vacías: cada una espera su timeout completo.
	for range 3 {
		app.AdvanceTripOfferTimeout(svc, trip)
		tripsRepo.AssertNotCalled(t, "UpdateTripStatus", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	}

	app.AdvanceTripOfferTimeout(svc, trip)
	tripsRepo.AssertExpectations(t)
}

func TestOfferTrip_NoPickupCellExpiresImmediately(t *testing.T) {
	repo := mocksriders.NewRidersRepo(t)
	stream := mocksriders.NewRidersStream(t)
	bcv := mockbcvports.NewBcvApp(t)
	tripsRepo := mockstrips.NewTripsRepo(t)
	tripEarnings := mocksriders.NewTripEarningRecorder(t)
	svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings, nil, nil, nil)
	matcher := asMatcher(t, svc)

	trip := samplePickupTrip(t)
	trip.PickupH3Cell = nil
	tripsRepo.On("UpdateTripStatus", mock.Anything, trip.ID, tripsModels.TripStatusSearching, tripsModels.TripStatusNoDriver).Return(true, nil)
	tripsRepo.On("SaveTripCancelReason", mock.Anything, trip.ID, tripsModels.CancelReasonNoDriver).Return(nil)

	matcher.OfferTrip(context.Background(), trip)

	tripsRepo.AssertExpectations(t)
}

func TestAcceptTrip_Success(t *testing.T) {
	repo := mocksriders.NewRidersRepo(t)
	stream := mocksriders.NewRidersStream(t)
	bcv := mockbcvports.NewBcvApp(t)
	tripsRepo := mockstrips.NewTripsRepo(t)
	tripEarnings := mocksriders.NewTripEarningRecorder(t)
	svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings, nil, nil, nil)
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
	tripsRepo.On("AssignTripRider", mock.Anything, trip.ID, riderID, mock.Anything).Return(true, nil)
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
	svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings, nil, nil, nil)
	matcher := asMatcher(t, svc)

	repo.On("SearchActiveOrdersByRiderID", mock.Anything, mock.Anything).Return(nil, errcustom.ErrNotFound)
	repo.On("SearchAvailableOrders", mock.Anything).Return(nil, errcustom.ErrNotFound)

	riderID := uuid.New()
	conn := mocksriders.NewRiderConn(t)
	conn.On("Send", mock.Anything).Return()
	require.NoError(t, svc.Register(context.Background(), riderID, conn, riderModels.RiderModeTrips))
	require.NoError(t, svc.UpdateLocation(context.Background(), riderID, 10.4806, -66.9036))

	trip := samplePickupTrip(t)
	otherRider := uuid.New()
	tripsRepo.On("GetTripByID", mock.Anything, trip.ID).Return(&tripsModels.Trip{ID: trip.ID, Status: tripsModels.TripStatusSearching}, nil).Once()
	tripsRepo.On("GetTripByID", mock.Anything, trip.ID).Return(&tripsModels.Trip{ID: trip.ID, Status: tripsModels.TripStatusAccepted, RiderID: &otherRider}, nil)
	tripsRepo.On("AssignTripRider", mock.Anything, trip.ID, riderID, mock.Anything).Return(false, nil)
	matcher.OfferTrip(context.Background(), trip)

	err := svc.AcceptTrip(context.Background(), riderID, trip.ID, conn)

	require.NoError(t, err)
	conn.AssertCalled(t, "Send", mock.MatchedBy(func(raw []byte) bool { return messageType(t, raw) == "trip_taken" }))
}

func TestAcceptTrip_RejectsWhenNotSearching(t *testing.T) {
	repo := mocksriders.NewRidersRepo(t)
	stream := mocksriders.NewRidersStream(t)
	bcv := mockbcvports.NewBcvApp(t)
	tripsRepo := mockstrips.NewTripsRepo(t)
	tripEarnings := mocksriders.NewTripEarningRecorder(t)
	svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings, nil, nil, nil)

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
	svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings, nil, nil, nil)

	riderID, tripID := uuid.New(), uuid.New()
	conn := mocksriders.NewRiderConn(t)
	conn.On("Send", mock.Anything).Return()

	tripsRepo.On("GetTripByID", mock.Anything, tripID).Return(&tripsModels.Trip{ID: tripID, RiderID: &riderID, Status: tripsModels.TripStatusAccepted}, nil)
	tripsRepo.On("UpdateTripStatus", mock.Anything, tripID, tripsModels.TripStatusAccepted, tripsModels.TripStatusArriving).Return(true, nil)

	err := svc.AdvanceTrip(context.Background(), riderID, tripID, "", conn)

	require.NoError(t, err)
}

func TestAdvanceTrip_ArrivingMovesToInProgress(t *testing.T) {
	repo := mocksriders.NewRidersRepo(t)
	stream := mocksriders.NewRidersStream(t)
	bcv := mockbcvports.NewBcvApp(t)
	tripsRepo := mockstrips.NewTripsRepo(t)
	tripEarnings := mocksriders.NewTripEarningRecorder(t)
	svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings, nil, nil, nil)

	riderID, tripID := uuid.New(), uuid.New()
	conn := mocksriders.NewRiderConn(t)
	var sent []byte
	conn.On("Send", mock.Anything).Run(func(args mock.Arguments) {
		sent = args.Get(0).([]byte)
	}).Return()

	pickup := "1234"
	tripsRepo.On("GetTripByID", mock.Anything, tripID).Return(&tripsModels.Trip{ID: tripID, RiderID: &riderID, Status: tripsModels.TripStatusArriving, PickupCode: &pickup}, nil)
	tripsRepo.On("StartTripWithPickupCode", mock.Anything, tripID, riderID, pickup, mock.Anything).Return(true, nil)

	err := svc.AdvanceTrip(context.Background(), riderID, tripID, pickup, conn)

	require.NoError(t, err)
	var msg struct {
		Trip struct {
			Status    string     `json:"status"`
			StartedAt *time.Time `json:"started_at"`
		} `json:"trip"`
	}
	require.NoError(t, json.Unmarshal(sent, &msg))
	assert.Equal(t, string(tripsModels.TripStatusInProgress), msg.Trip.Status)
	assert.NotNil(t, msg.Trip.StartedAt)
}

func TestAdvanceTrip_InProgressCompletesAndRecordsEarning(t *testing.T) {
	repo := mocksriders.NewRidersRepo(t)
	stream := mocksriders.NewRidersStream(t)
	bcv := mockbcvports.NewBcvApp(t)
	tripsRepo := mockstrips.NewTripsRepo(t)
	tripEarnings := mocksriders.NewTripEarningRecorder(t)
	svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings, nil, nil, nil)

	riderID, tripID := uuid.New(), uuid.New()
	conn := mocksriders.NewRiderConn(t)
	conn.On("Send", mock.Anything).Return()

	// completeTrip usa el fare ya fijado en trip.Fare, no lo recalcula.
	fare := deliveryfee.DevPlaceholderFee
	tripsRepo.On("GetTripByID", mock.Anything, tripID).Return(&tripsModels.Trip{ID: tripID, RiderID: &riderID, Status: tripsModels.TripStatusInProgress, Fare: &fare}, nil)
	tripsRepo.On("CompleteTrip", mock.Anything, tripID, deliveryfee.DevPlaceholderFee, "5678").Return(true, nil)
	bcv.On("GetRate", mock.Anything).Return(&bcvmodels.ExchangeRate{Rate: 40}, nil)
	tripEarnings.On("RecordTripEarning", mock.Anything, riderID, tripID, deliveryfee.DevPlaceholderFee, 40.0).Return(nil)

	err := svc.AdvanceTrip(context.Background(), riderID, tripID, "5678", conn)

	require.NoError(t, err)
}

func TestAdvanceTrip_RejectsWhenRiderNotAssigned(t *testing.T) {
	repo := mocksriders.NewRidersRepo(t)
	stream := mocksriders.NewRidersStream(t)
	bcv := mockbcvports.NewBcvApp(t)
	tripsRepo := mockstrips.NewTripsRepo(t)
	tripEarnings := mocksriders.NewTripEarningRecorder(t)
	svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings, nil, nil, nil)

	assignedRiderID, otherRiderID, tripID := uuid.New(), uuid.New(), uuid.New()
	tripsRepo.On("GetTripByID", mock.Anything, tripID).Return(&tripsModels.Trip{ID: tripID, RiderID: &assignedRiderID, Status: tripsModels.TripStatusAccepted}, nil)

	err := svc.AdvanceTrip(context.Background(), otherRiderID, tripID, "", mocksriders.NewRiderConn(t))

	assert.ErrorIs(t, err, errcustom.ErrInvalidTransition)
}

func TestAdvanceTrip_RejectsTerminalStatus(t *testing.T) {
	repo := mocksriders.NewRidersRepo(t)
	stream := mocksriders.NewRidersStream(t)
	bcv := mockbcvports.NewBcvApp(t)
	tripsRepo := mockstrips.NewTripsRepo(t)
	tripEarnings := mocksriders.NewTripEarningRecorder(t)
	svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings, nil, nil, nil)

	riderID, tripID := uuid.New(), uuid.New()
	tripsRepo.On("GetTripByID", mock.Anything, tripID).Return(&tripsModels.Trip{ID: tripID, RiderID: &riderID, Status: tripsModels.TripStatusCompleted}, nil)

	err := svc.AdvanceTrip(context.Background(), riderID, tripID, "", mocksriders.NewRiderConn(t))

	assert.ErrorIs(t, err, errcustom.ErrInvalidTransition)
}

// TestAdvanceTrip_ConcurrentCompletionIsSerialized cubre la vulnerabilidad de
// seguridad #1 (auditoría): sin el guard completingTrips, un spam de
// advance_trip sobre el mismo trip en in_progress podía disparar varias
// llamadas en paralelo a CompleteTrip/RecordTripEarning antes de que la
// primera confirmara en DB. Aquí se verifica que la segunda llamada, mientras
// la primera sigue "en vuelo" dentro de CompleteTrip, se rechaza de inmediato
// sin volver a invocar CompleteTrip (el mock .Once() lo garantiza).
func TestAdvanceTrip_ConcurrentCompletionIsSerialized(t *testing.T) {
	repo := mocksriders.NewRidersRepo(t)
	stream := mocksriders.NewRidersStream(t)
	bcv := mockbcvports.NewBcvApp(t)
	tripsRepo := mockstrips.NewTripsRepo(t)
	tripEarnings := mocksriders.NewTripEarningRecorder(t)
	svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings, nil, nil, nil)

	riderID, tripID := uuid.New(), uuid.New()
	conn := mocksriders.NewRiderConn(t)
	conn.On("Send", mock.Anything).Return()

	entered := make(chan struct{})
	proceed := make(chan struct{})

	fare := deliveryfee.DevPlaceholderFee
	tripsRepo.On("GetTripByID", mock.Anything, tripID).Return(&tripsModels.Trip{ID: tripID, RiderID: &riderID, Status: tripsModels.TripStatusInProgress, Fare: &fare}, nil)
	tripsRepo.On("CompleteTrip", mock.Anything, tripID, deliveryfee.DevPlaceholderFee, "5678").Run(func(mock.Arguments) {
		close(entered)
		<-proceed
	}).Return(true, nil).Once()
	bcv.On("GetRate", mock.Anything).Return(&bcvmodels.ExchangeRate{Rate: 40}, nil)
	tripEarnings.On("RecordTripEarning", mock.Anything, riderID, tripID, deliveryfee.DevPlaceholderFee, 40.0).Return(nil)

	var wg sync.WaitGroup
	var firstErr error
	wg.Go(func() {
		firstErr = svc.AdvanceTrip(context.Background(), riderID, tripID, "5678", conn)
	})

	<-entered // la primera llamada ya está dentro de CompleteTrip

	secondErr := svc.AdvanceTrip(context.Background(), riderID, tripID, "5678", conn)
	close(proceed)
	wg.Wait()

	assert.ErrorIs(t, secondErr, errcustom.ErrInvalidTransition)
	require.NoError(t, firstErr)
	tripsRepo.AssertExpectations(t) // CompleteTrip solo se llamó una vez (.Once())
}

func TestCancelAcceptedTrip_ReoffersExcludingCancellingRider(t *testing.T) {
	for _, status := range []tripsModels.TripStatus{tripsModels.TripStatusAccepted, tripsModels.TripStatusArriving} {
		t.Run(string(status), func(t *testing.T) {
			repo := mocksriders.NewRidersRepo(t)
			stream := mocksriders.NewRidersStream(t)
			bcv := mockbcvports.NewBcvApp(t)
			tripsRepo := mockstrips.NewTripsRepo(t)
			tripEarnings := mocksriders.NewTripEarningRecorder(t)
			svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings, nil, nil, nil)

			repo.On("SearchActiveOrdersByRiderID", mock.Anything, mock.Anything).Return(nil, errcustom.ErrNotFound)
			repo.On("SearchAvailableOrders", mock.Anything).Return(nil, errcustom.ErrNotFound)

			riderA, riderB := uuid.New(), uuid.New()
			connA, connB := mocksriders.NewRiderConn(t), mocksriders.NewRiderConn(t)
			require.NoError(t, svc.Register(context.Background(), riderA, connA, riderModels.RiderModeTrips))
			require.NoError(t, svc.Register(context.Background(), riderB, connB, riderModels.RiderModeTrips))
			require.NoError(t, svc.UpdateLocation(context.Background(), riderA, 10.4807, -66.9036))
			require.NoError(t, svc.UpdateLocation(context.Background(), riderB, 10.4820, -66.9036))

			trip := samplePickupTrip(t)
			trip.RiderID = &riderA
			trip.Status = status
			tripsRepo.On("GetTripByID", mock.Anything, trip.ID).Return(&trip, nil)
			tripsRepo.On("ReleaseTripRider", mock.Anything, trip.ID, riderA).Return(true, nil)

			var statusToA, offerToA, offerToB bool
			connA.On("Send", mock.Anything).Run(func(args mock.Arguments) {
				kind := messageType(t, args.Get(0).([]byte))
				if kind == "trip_status" {
					statusToA = true
				}
				if kind == "trip_offer" {
					offerToA = true
				}
			}).Return()
			connB.On("Send", mock.Anything).Run(func(args mock.Arguments) {
				if messageType(t, args.Get(0).([]byte)) == "trip_offer" {
					offerToB = true
				}
			}).Return()

			err := svc.CancelAcceptedTrip(context.Background(), riderA, trip.ID, connA)

			require.NoError(t, err)
			require.True(t, statusToA)
			require.True(t, offerToB)
			require.False(t, offerToA)
		})
	}
}

func TestCancelAcceptedTrip_NoOtherRidersKeepsSearching(t *testing.T) {
	repo := mocksriders.NewRidersRepo(t)
	stream := mocksriders.NewRidersStream(t)
	bcv := mockbcvports.NewBcvApp(t)
	tripsRepo := mockstrips.NewTripsRepo(t)
	tripEarnings := mocksriders.NewTripEarningRecorder(t)
	svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings, nil, nil, nil)

	repo.On("SearchActiveOrdersByRiderID", mock.Anything, mock.Anything).Return(nil, errcustom.ErrNotFound)
	repo.On("SearchAvailableOrders", mock.Anything).Return(nil, errcustom.ErrNotFound)

	riderA := uuid.New()
	connA := mocksriders.NewRiderConn(t)
	require.NoError(t, svc.Register(context.Background(), riderA, connA, riderModels.RiderModeTrips))
	require.NoError(t, svc.UpdateLocation(context.Background(), riderA, 10.4807, -66.9036))

	trip := samplePickupTrip(t)
	t.Cleanup(func() { app.StopTripOfferTimerForTest(svc, trip.ID) })
	trip.RiderID = &riderA
	trip.Status = tripsModels.TripStatusAccepted
	tripsRepo.On("GetTripByID", mock.Anything, trip.ID).Return(&trip, nil)
	tripsRepo.On("ReleaseTripRider", mock.Anything, trip.ID, riderA).Return(true, nil)
	connA.On("Send", mock.Anything).Return()

	err := svc.CancelAcceptedTrip(context.Background(), riderA, trip.ID, connA)

	require.NoError(t, err)
	tripsRepo.AssertNotCalled(t, "UpdateTripStatus", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestCancelAcceptedTrip_RejectsWhenNotAssigned(t *testing.T) {
	repo := mocksriders.NewRidersRepo(t)
	stream := mocksriders.NewRidersStream(t)
	bcv := mockbcvports.NewBcvApp(t)
	tripsRepo := mockstrips.NewTripsRepo(t)
	tripEarnings := mocksriders.NewTripEarningRecorder(t)
	svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings, nil, nil, nil)

	riderA, other := uuid.New(), uuid.New()
	tripID := uuid.New()
	tripsRepo.On("GetTripByID", mock.Anything, tripID).Return(&tripsModels.Trip{
		ID: tripID, RiderID: &other, Status: tripsModels.TripStatusAccepted,
	}, nil)

	err := svc.CancelAcceptedTrip(context.Background(), riderA, tripID, mocksriders.NewRiderConn(t))

	assert.ErrorIs(t, err, errcustom.ErrInvalidTransition)
	tripsRepo.AssertNotCalled(t, "ReleaseTripRider", mock.Anything, mock.Anything, mock.Anything)
}

func TestCancelAcceptedTrip_RejectsAfterPickup(t *testing.T) {
	for _, status := range []tripsModels.TripStatus{tripsModels.TripStatusInProgress, tripsModels.TripStatusCompleted} {
		t.Run(string(status), func(t *testing.T) {
			repo := mocksriders.NewRidersRepo(t)
			stream := mocksriders.NewRidersStream(t)
			bcv := mockbcvports.NewBcvApp(t)
			tripsRepo := mockstrips.NewTripsRepo(t)
			tripEarnings := mocksriders.NewTripEarningRecorder(t)
			svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings, nil, nil, nil)

			riderA := uuid.New()
			tripID := uuid.New()
			tripsRepo.On("GetTripByID", mock.Anything, tripID).Return(&tripsModels.Trip{
				ID: tripID, RiderID: &riderA, Status: status,
			}, nil)

			err := svc.CancelAcceptedTrip(context.Background(), riderA, tripID, mocksriders.NewRiderConn(t))

			assert.ErrorIs(t, err, errcustom.ErrInvalidTransition)
			tripsRepo.AssertNotCalled(t, "ReleaseTripRider", mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

func messageType(t *testing.T, raw []byte) string {
	t.Helper()
	var msg struct {
		Type string `json:"type"`
	}
	require.NoError(t, json.Unmarshal(raw, &msg))
	return msg.Type
}

func newTripOfferSvc(t *testing.T) (ridersPorts.RidersApp, *mockstrips.TripsRepo, *mocksriders.RidersRepo) {
	t.Helper()
	repo := mocksriders.NewRidersRepo(t)
	stream := mocksriders.NewRidersStream(t)
	bcv := mockbcvports.NewBcvApp(t)
	tripsRepo := mockstrips.NewTripsRepo(t)
	tripEarnings := mocksriders.NewTripEarningRecorder(t)
	return app.New(repo, stream, bcv, tripsRepo, tripEarnings, nil, nil, nil), tripsRepo, repo
}

func countTripOffers(t *testing.T) (map[uuid.UUID]int, func(uuid.UUID, *mocksriders.RiderConn)) {
	t.Helper()
	offers := map[uuid.UUID]int{}
	var mu sync.Mutex
	watch := func(id uuid.UUID, conn *mocksriders.RiderConn) {
		conn.On("Send", mock.Anything).Run(func(args mock.Arguments) {
			if messageType(t, args.Get(0).([]byte)) != "trip_offer" {
				return
			}
			mu.Lock()
			offers[id]++
			mu.Unlock()
		}).Return().Maybe()
	}
	return offers, watch
}

func latLngAtRing(t *testing.T, origin string, ring int) (float64, float64) {
	t.Helper()
	outer, err := geo.GridDisk(origin, ring)
	require.NoError(t, err)
	inner, err := geo.GridDisk(origin, ring-1)
	require.NoError(t, err)
	seen := make(map[string]struct{}, len(inner))
	for _, cell := range inner {
		seen[cell] = struct{}{}
	}
	for _, cell := range outer {
		if _, ok := seen[cell]; ok {
			continue
		}
		lat, lng, err := geo.CellToLatLng(cell)
		require.NoError(t, err)
		return lat, lng
	}
	t.Fatalf("no hay celda en el anillo %d", ring)
	return 0, 0
}

func TestOfferTrip_SecondRoundExcludesFirstCandidatesAndExpandsRing(t *testing.T) {
	svc, tripsRepo, repo := newTripOfferSvc(t)
	trip := samplePickupTrip(t)
	t.Cleanup(func() { app.StopTripOfferTimerForTest(svc, trip.ID) })
	repo.On("SearchActiveOrdersByRiderID", mock.Anything, mock.Anything).Return(nil, errcustom.ErrNotFound)
	repo.On("SearchAvailableOrders", mock.Anything).Return(nil, errcustom.ErrNotFound)

	offers, watch := countTripOffers(t)
	near := make([]uuid.UUID, 3)
	for i := range near {
		near[i] = uuid.New()
		conn := mocksriders.NewRiderConn(t)
		watch(near[i], conn)
		require.NoError(t, svc.Register(context.Background(), near[i], conn, riderModels.RiderModeTrips))
		require.NoError(t, svc.UpdateLocation(context.Background(), near[i], 10.4806+float64(i)*0.0001, -66.9036))
	}
	farID := uuid.New()
	farLat, farLng := latLngAtRing(t, *trip.PickupH3Cell, 4)
	farConn := mocksriders.NewRiderConn(t)
	watch(farID, farConn)
	require.NoError(t, svc.Register(context.Background(), farID, farConn, riderModels.RiderModeTrips))
	require.NoError(t, svc.UpdateLocation(context.Background(), farID, farLat, farLng))

	searching := trip
	searching.Status = tripsModels.TripStatusSearching
	tripsRepo.On("GetTripByID", mock.Anything, trip.ID).Return(&searching, nil)

	asMatcher(t, svc).OfferTrip(context.Background(), trip)
	for _, id := range near {
		require.Equal(t, 1, offers[id])
	}
	require.Zero(t, offers[farID])

	app.AdvanceTripOfferTimeout(svc, trip)
	for _, id := range near {
		require.Equal(t, 1, offers[id])
	}
	require.Equal(t, 1, offers[farID])
	tripsRepo.AssertNotCalled(t, "UpdateTripStatus", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestOfferTrip_EmptyNextRoundWaitsThenExpiresAtRoundCap(t *testing.T) {
	svc, tripsRepo, repo := newTripOfferSvc(t)
	trip := samplePickupTrip(t)
	t.Cleanup(func() { app.StopTripOfferTimerForTest(svc, trip.ID) })
	repo.On("SearchActiveOrdersByRiderID", mock.Anything, mock.Anything).Return(nil, errcustom.ErrNotFound)
	repo.On("SearchAvailableOrders", mock.Anything).Return(nil, errcustom.ErrNotFound)

	offers, watch := countTripOffers(t)
	ids := make([]uuid.UUID, 3)
	for i := range ids {
		ids[i] = uuid.New()
		conn := mocksriders.NewRiderConn(t)
		watch(ids[i], conn)
		require.NoError(t, svc.Register(context.Background(), ids[i], conn, riderModels.RiderModeTrips))
		require.NoError(t, svc.UpdateLocation(context.Background(), ids[i], 10.4806+float64(i)*0.0001, -66.9036))
	}

	searching := trip
	searching.Status = tripsModels.TripStatusSearching
	tripsRepo.On("GetTripByID", mock.Anything, trip.ID).Return(&searching, nil)
	tripsRepo.On("UpdateTripStatus", mock.Anything, trip.ID, tripsModels.TripStatusSearching, tripsModels.TripStatusNoDriver).Return(true, nil).Once()
	tripsRepo.On("SaveTripCancelReason", mock.Anything, trip.ID, mock.Anything).Return(nil).Maybe()

	asMatcher(t, svc).OfferTrip(context.Background(), trip)

	// Ronda 2 sin candidatos nuevos: espera, no expira; 3 y 4 igual.
	for range 3 {
		app.AdvanceTripOfferTimeout(svc, trip)
		tripsRepo.AssertNotCalled(t, "UpdateTripStatus", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	}

	app.AdvanceTripOfferTimeout(svc, trip)
	tripsRepo.AssertNumberOfCalls(t, "UpdateTripStatus", 1)

	for _, id := range ids {
		require.Equal(t, 1, offers[id])
	}
}

func TestOfferTrip_ExpiresAtRoundCapEvenIfCandidatesRemain(t *testing.T) {
	svc, tripsRepo, repo := newTripOfferSvc(t)
	trip := samplePickupTrip(t)
	t.Cleanup(func() { app.StopTripOfferTimerForTest(svc, trip.ID) })
	repo.On("SearchActiveOrdersByRiderID", mock.Anything, mock.Anything).Return(nil, errcustom.ErrNotFound)
	repo.On("SearchAvailableOrders", mock.Anything).Return(nil, errcustom.ErrNotFound)

	offers, watch := countTripOffers(t)
	ids := make([]uuid.UUID, 13)
	for i := range ids {
		ids[i] = uuid.New()
		conn := mocksriders.NewRiderConn(t)
		watch(ids[i], conn)
		require.NoError(t, svc.Register(context.Background(), ids[i], conn, riderModels.RiderModeTrips))
		require.NoError(t, svc.UpdateLocation(context.Background(), ids[i], 10.4806+float64(i)*0.00005, -66.9036))
	}

	searching := trip
	searching.Status = tripsModels.TripStatusSearching
	tripsRepo.On("GetTripByID", mock.Anything, trip.ID).Return(&searching, nil)
	tripsRepo.On("UpdateTripStatus", mock.Anything, trip.ID, tripsModels.TripStatusSearching, tripsModels.TripStatusNoDriver).Return(true, nil).Once()
	tripsRepo.On("SaveTripCancelReason", mock.Anything, trip.ID, mock.Anything).Return(nil).Maybe()

	asMatcher(t, svc).OfferTrip(context.Background(), trip)
	for range 3 {
		app.AdvanceTripOfferTimeout(svc, trip)
	}
	offered := 0
	for _, id := range ids {
		offered += offers[id]
	}
	require.Equal(t, 12, offered)

	app.AdvanceTripOfferTimeout(svc, trip)
	offered = 0
	for _, id := range ids {
		offered += offers[id]
		require.LessOrEqual(t, offers[id], 1)
	}
	require.Equal(t, 12, offered)
}

type recordingCustomer struct {
	trips     []tripsModels.Trip
	locations int
	lastTrip  uuid.UUID
}

func (r *recordingCustomer) NotifyTrip(_ context.Context, trip tripsModels.Trip) {
	r.trips = append(r.trips, trip)
}

func (r *recordingCustomer) NotifyRiderLocation(_ context.Context, _, tripID uuid.UUID, _, _ float64) {
	r.locations++
	r.lastTrip = tripID
}

func TestAcceptTrip_NotifiesCustomerCodeHiddenFromRider(t *testing.T) {
	repo := mocksriders.NewRidersRepo(t)
	stream := mocksriders.NewRidersStream(t)
	bcv := mockbcvports.NewBcvApp(t)
	tripsRepo := mockstrips.NewTripsRepo(t)
	tripEarnings := mocksriders.NewTripEarningRecorder(t)
	customer := &recordingCustomer{}
	svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings, customer, nil, nil)
	matcher := asMatcher(t, svc)

	repo.On("SearchActiveOrdersByRiderID", mock.Anything, mock.Anything).Return(nil, errcustom.ErrNotFound)
	repo.On("SearchAvailableOrders", mock.Anything).Return(nil, errcustom.ErrNotFound)

	riderID := uuid.New()
	customerID := uuid.New()
	var sent [][]byte
	conn := mocksriders.NewRiderConn(t)
	conn.On("Send", mock.Anything).Run(func(args mock.Arguments) {
		sent = append(sent, append([]byte(nil), args.Get(0).([]byte)...))
	}).Return()
	require.NoError(t, svc.Register(context.Background(), riderID, conn, riderModels.RiderModeTrips))
	require.NoError(t, svc.UpdateLocation(context.Background(), riderID, 10.4806, -66.9036))

	trip := samplePickupTrip(t)
	trip.CustomerID = customerID
	tripsRepo.On("GetTripByID", mock.Anything, trip.ID).Return(&tripsModels.Trip{
		ID: trip.ID, CustomerID: customerID, Status: tripsModels.TripStatusSearching,
	}, nil)
	tripsRepo.On("AssignTripRider", mock.Anything, trip.ID, riderID, mock.Anything).Return(true, nil)
	matcher.OfferTrip(context.Background(), trip)

	require.NoError(t, svc.AcceptTrip(context.Background(), riderID, trip.ID, conn))

	require.Len(t, customer.trips, 1)
	require.NotNil(t, customer.trips[0].PickupCode)
	require.Len(t, *customer.trips[0].PickupCode, 4)
	for _, msg := range sent {
		require.NotContains(t, string(msg), "pickup_code")
	}

	require.NoError(t, svc.UpdateLocation(context.Background(), riderID, 10.49, -66.91))
	require.Equal(t, 1, customer.locations)
	require.Equal(t, trip.ID, customer.lastTrip)
}

func TestAdvanceTrip_WrongPickupCodeDoesNotStartTrip(t *testing.T) {
	repo := mocksriders.NewRidersRepo(t)
	stream := mocksriders.NewRidersStream(t)
	bcv := mockbcvports.NewBcvApp(t)
	tripsRepo := mockstrips.NewTripsRepo(t)
	tripEarnings := mocksriders.NewTripEarningRecorder(t)
	svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings, nil, nil, nil)

	riderID, tripID := uuid.New(), uuid.New()
	pickup := "1234"
	tripsRepo.On("GetTripByID", mock.Anything, tripID).Return(&tripsModels.Trip{
		ID: tripID, RiderID: &riderID, Status: tripsModels.TripStatusArriving, PickupCode: &pickup,
	}, nil)
	tripsRepo.On("StartTripWithPickupCode", mock.Anything, tripID, riderID, "0000", mock.Anything).Return(false, nil)

	err := svc.AdvanceTrip(context.Background(), riderID, tripID, "0000", mocksriders.NewRiderConn(t))

	require.ErrorIs(t, err, errcustom.ErrInvalidCode)
}
