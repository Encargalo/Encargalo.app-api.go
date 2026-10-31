package app_test

import (
	"context"
	"testing"
	"time"

	mockbcvports "Encargalo.app-api.go/internal/mocks/bcv/domain/ports"
	mocksriders "Encargalo.app-api.go/internal/mocks/riders/domain/ports"
	mockstrips "Encargalo.app-api.go/internal/mocks/trips/domain/ports"
	ordersModels "Encargalo.app-api.go/internal/orders/domain/models"
	"Encargalo.app-api.go/internal/riders/app"
	riderModels "Encargalo.app-api.go/internal/riders/domain/models"
	"Encargalo.app-api.go/internal/shared/errcustom"
	tripsModels "Encargalo.app-api.go/internal/trips/domain/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestOfferTrip_NotifiesPushWithLiveCandidate verifica que la ronda de oferta
// manda el push al mismo candidato que recibe el WS — el push nunca amplía el
// pool, solo lo acompaña. NotifyTripOffer corre en una goroutine
// (fire-and-forget), así que la aserción espera por un canal en vez de leer
// el mock inmediatamente después de llamar a OfferTrip.
func TestOfferTrip_NotifiesPushWithLiveCandidate(t *testing.T) {
	repo := mocksriders.NewRidersRepo(t)
	stream := newRidersStream(t)
	bcv := mockbcvports.NewBcvApp(t)
	tripsRepo := mockstrips.NewTripsRepo(t)
	tripEarnings := mocksriders.NewTripEarningRecorder(t)
	presence := newPresenceStore(t)
	push := mocksriders.NewPushNotifier(t)

	svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings, nil, nil, nil, nil, presence, newOfferTimeoutStore(t), nil, push, nil, nil, nil, nil)

	repo.On("SearchActiveOrdersByRiderID", mock.Anything, mock.Anything).Return(nil, errcustom.ErrNotFound)
	repo.On("SearchAvailableOrders", mock.Anything).Return(nil, errcustom.ErrNotFound)

	trip := samplePickupTrip(t)
	t.Cleanup(func() { app.StopTripOfferTimerForTest(svc, trip.ID) })
	riderID, conn := placeTripRider(t, svc, trip.PickupLatitude, trip.PickupLongitude)
	conn.On("Send", mock.Anything).Return()

	searching := trip
	searching.Status = tripsModels.TripStatusSearching
	tripsRepo.On("GetTripByID", mock.Anything, trip.ID).Return(&searching, nil).Maybe()

	done := make(chan []uuid.UUID, 1)
	push.On("NotifyTripOffer", mock.Anything, mock.Anything, mock.MatchedBy(func(tr tripsModels.Trip) bool { return tr.ID == trip.ID })).
		Run(func(args mock.Arguments) { done <- args.Get(1).([]uuid.UUID) }).
		Return()

	asMatcher(t, svc).OfferTrip(context.Background(), trip)

	select {
	case ids := <-done:
		require.Contains(t, ids, riderID)
	case <-time.After(2 * time.Second):
		t.Fatal("NotifyTripOffer no se llamó a tiempo")
	}
}

// TestBroadcastNewOrder_NotifiesPushWithConnectedRider cubre el pool de
// pedidos: a diferencia de viajes, no hay radio geográfico — el push va al
// mismo universo de riders conectados en modo pedidos que ya recibe el WS.
func TestBroadcastNewOrder_NotifiesPushWithConnectedRider(t *testing.T) {
	repo := mocksriders.NewRidersRepo(t)
	stream := newRidersStream(t)
	bcv := mockbcvports.NewBcvApp(t)
	tripsRepo := mockstrips.NewTripsRepo(t)
	tripEarnings := mocksriders.NewTripEarningRecorder(t)
	push := mocksriders.NewPushNotifier(t)

	svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings, nil, nil, nil, nil, newPresenceStore(t), newOfferTimeoutStore(t), nil, push, nil, nil, nil, nil)

	repo.On("SearchActiveOrdersByRiderID", mock.Anything, mock.Anything).Return(nil, errcustom.ErrNotFound)
	repo.On("SearchAvailableOrders", mock.Anything).Return(nil, errcustom.ErrNotFound)

	riderID := uuid.New()
	conn := mocksriders.NewRiderConn(t)
	conn.On("Send", mock.Anything).Return()
	require.NoError(t, svc.Register(context.Background(), riderID, conn, riderModels.RiderModeOrders))

	order := ordersModels.Order{ID: uuid.New(), ShopID: uuid.New(), Status: string(ordersModels.StatusReady)}
	repo.On("SearchShopByID", mock.Anything, order.ShopID).Return(&ordersModels.ShopOrder{}, nil)
	repo.On("SearchOrdersByID", mock.Anything, order.ID).Return(&order, nil).Maybe()

	done := make(chan []uuid.UUID, 1)
	push.On("NotifyNewOrder", mock.Anything, mock.Anything, mock.MatchedBy(func(o ordersModels.Order) bool { return o.ID == order.ID })).
		Run(func(args mock.Arguments) { done <- args.Get(1).([]uuid.UUID) }).
		Return()

	asBroadcaster(t, svc).BroadcastNewOrder(context.Background(), order)

	select {
	case ids := <-done:
		require.Contains(t, ids, riderID)
	case <-time.After(2 * time.Second):
		t.Fatal("NotifyNewOrder no se llamó a tiempo")
	}
}

// TestNotifyTripCancelled_NotifiesPushToAssignedRiderOnly verifica que la
// cancelación por el cliente manda push solo al rider ya asignado — el canal
// que lo saca del WS en segundo plano, no solo del socket.
func TestNotifyTripCancelled_NotifiesPushToAssignedRiderOnly(t *testing.T) {
	repo := mocksriders.NewRidersRepo(t)
	stream := newRidersStream(t)
	bcv := mockbcvports.NewBcvApp(t)
	tripsRepo := mockstrips.NewTripsRepo(t)
	tripEarnings := mocksriders.NewTripEarningRecorder(t)
	push := mocksriders.NewPushNotifier(t)

	svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings, nil, nil, nil, nil, newPresenceStore(t), newOfferTimeoutStore(t), nil, push, nil, nil, nil, nil)

	repo.On("SearchActiveOrdersByRiderID", mock.Anything, mock.Anything).Return(nil, errcustom.ErrNotFound)
	repo.On("SearchAvailableOrders", mock.Anything).Return(nil, errcustom.ErrNotFound)

	trip := samplePickupTrip(t)
	t.Cleanup(func() { app.StopTripOfferTimerForTest(svc, trip.ID) })
	riderID, conn := placeTripRider(t, svc, trip.PickupLatitude, trip.PickupLongitude)
	conn.On("Send", mock.Anything).Return()

	push.On("NotifyTripOffer", mock.Anything, mock.Anything, mock.Anything).Maybe().Return()
	tripsRepo.On("GetTripByID", mock.Anything, trip.ID).Return(&tripsModels.Trip{
		ID: trip.ID, Status: tripsModels.TripStatusSearching,
	}, nil)
	tripsRepo.On("AssignTripRider", mock.Anything, trip.ID, riderID, mock.Anything, mock.Anything).Return(true, nil)
	asMatcher(t, svc).OfferTrip(context.Background(), trip)
	require.NoError(t, svc.AcceptTrip(context.Background(), riderID, trip.ID, conn))

	done := make(chan []uuid.UUID, 1)
	push.On("NotifyTripCancelled", mock.Anything, mock.Anything, mock.MatchedBy(func(tr tripsModels.Trip) bool { return tr.ID == trip.ID })).
		Run(func(args mock.Arguments) { done <- args.Get(1).([]uuid.UUID) }).
		Return()

	cancelled := trip
	cancelled.RiderID = &riderID
	cancelled.Status = tripsModels.TripStatusCancelled
	asTripNotifier(t, svc).NotifyTripCancelled(context.Background(), cancelled)

	select {
	case ids := <-done:
		require.Equal(t, []uuid.UUID{riderID}, ids)
	case <-time.After(2 * time.Second):
		t.Fatal("NotifyTripCancelled no se llamó a tiempo")
	}
}

// TestNotifyTripCancelled_NoPushWithoutAssignedRider cubre el viaje que
// todavía se estaba ofreciendo (nadie lo aceptó): perder la oferta no es algo
// que requiera avisar fuera del WS, así que no debe haber push.
func TestNotifyTripCancelled_NoPushWithoutAssignedRider(t *testing.T) {
	repo := mocksriders.NewRidersRepo(t)
	stream := newRidersStream(t)
	bcv := mockbcvports.NewBcvApp(t)
	tripsRepo := mockstrips.NewTripsRepo(t)
	tripEarnings := mocksriders.NewTripEarningRecorder(t)
	push := mocksriders.NewPushNotifier(t)

	svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings, nil, nil, nil, nil, newPresenceStore(t), newOfferTimeoutStore(t), nil, push, nil, nil, nil, nil)

	repo.On("SearchActiveOrdersByRiderID", mock.Anything, mock.Anything).Return(nil, errcustom.ErrNotFound)
	repo.On("SearchAvailableOrders", mock.Anything).Return(nil, errcustom.ErrNotFound)

	trip := samplePickupTrip(t)
	t.Cleanup(func() { app.StopTripOfferTimerForTest(svc, trip.ID) })
	_, conn := placeTripRider(t, svc, trip.PickupLatitude, trip.PickupLongitude)
	conn.On("Send", mock.Anything).Return()
	push.On("NotifyTripOffer", mock.Anything, mock.Anything, mock.Anything).Maybe().Return()

	asMatcher(t, svc).OfferTrip(context.Background(), trip)

	cancelled := trip
	cancelled.Status = tripsModels.TripStatusCancelled
	asTripNotifier(t, svc).NotifyTripCancelled(context.Background(), cancelled)

	push.AssertNotCalled(t, "NotifyTripCancelled", mock.Anything, mock.Anything, mock.Anything)
}

// TestBroadcastOrderUpdate_NotifiesPushOnRejectToAssignedRider cubre el mismo
// hueco que NotifyTripCancelled pero para pedidos: al rider ya asignado a
// quien la tienda rechaza, no solo le llega el order_update por WS.
func TestBroadcastOrderUpdate_NotifiesPushOnRejectToAssignedRider(t *testing.T) {
	repo := mocksriders.NewRidersRepo(t)
	stream := newRidersStream(t)
	bcv := mockbcvports.NewBcvApp(t)
	tripsRepo := mockstrips.NewTripsRepo(t)
	tripEarnings := mocksriders.NewTripEarningRecorder(t)
	push := mocksriders.NewPushNotifier(t)

	svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings, nil, nil, nil, nil, newPresenceStore(t), newOfferTimeoutStore(t), nil, push, nil, nil, nil, nil)

	repo.On("SearchActiveOrdersByRiderID", mock.Anything, mock.Anything).Return(nil, errcustom.ErrNotFound)
	repo.On("SearchAvailableOrders", mock.Anything).Return(nil, errcustom.ErrNotFound)

	riderID := uuid.New()
	conn := mocksriders.NewRiderConn(t)
	conn.On("Send", mock.Anything).Return()
	require.NoError(t, svc.Register(context.Background(), riderID, conn, riderModels.RiderModeOrders))

	order := ordersModels.Order{ID: uuid.New(), ShopID: uuid.New(), CustomerID: uuid.New(), Status: string(ordersModels.StatusInPreparation)}
	repo.On("SearchShopByID", mock.Anything, order.ShopID).Return(&ordersModels.ShopOrder{}, nil)
	repo.On("SearchOrdersByID", mock.Anything, order.ID).Return(&order, nil)
	repo.On("AssignRider", mock.Anything, order.ID, riderID).Return(nil)
	bcv.On("GetRate", mock.Anything).Return(nil, errcustom.ErrUnexpectedError)
	push.On("NotifyNewOrder", mock.Anything, mock.Anything, mock.Anything).Maybe().Return()

	require.NoError(t, svc.AcceptOrder(context.Background(), riderID, order.ID, conn))

	done := make(chan []uuid.UUID, 1)
	push.On("NotifyOrderCancelled", mock.Anything, mock.Anything, mock.MatchedBy(func(o ordersModels.Order) bool { return o.ID == order.ID })).
		Run(func(args mock.Arguments) { done <- args.Get(1).([]uuid.UUID) }).
		Return()

	rejected := order
	rejected.Status = string(ordersModels.StatusRejected)
	asBroadcaster(t, svc).BroadcastOrderUpdate(context.Background(), rejected)

	select {
	case ids := <-done:
		require.Equal(t, []uuid.UUID{riderID}, ids)
	case <-time.After(2 * time.Second):
		t.Fatal("NotifyOrderCancelled no se llamó a tiempo")
	}
}

// TestBroadcastOrderUpdate_NoPushWithoutAssignedRider cubre el pedido que
// nadie había aceptado: perder la oferta no requiere avisar fuera del WS.
func TestBroadcastOrderUpdate_NoPushWithoutAssignedRider(t *testing.T) {
	repo := mocksriders.NewRidersRepo(t)
	stream := newRidersStream(t)
	bcv := mockbcvports.NewBcvApp(t)
	tripsRepo := mockstrips.NewTripsRepo(t)
	tripEarnings := mocksriders.NewTripEarningRecorder(t)
	push := mocksriders.NewPushNotifier(t)

	svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings, nil, nil, nil, nil, newPresenceStore(t), newOfferTimeoutStore(t), nil, push, nil, nil, nil, nil)

	order := ordersModels.Order{ID: uuid.New(), ShopID: uuid.New(), Status: string(ordersModels.StatusRejected)}
	repo.On("SearchShopByID", mock.Anything, order.ShopID).Return(&ordersModels.ShopOrder{}, nil)
	asBroadcaster(t, svc).BroadcastOrderUpdate(context.Background(), order)

	push.AssertNotCalled(t, "NotifyOrderCancelled", mock.Anything, mock.Anything, mock.Anything)
}
