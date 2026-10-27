package app_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	mockbcvports "Encargalo.app-api.go/internal/mocks/bcv/domain/ports"
	mocksriders "Encargalo.app-api.go/internal/mocks/riders/domain/ports"
	mockstrips "Encargalo.app-api.go/internal/mocks/trips/domain/ports"
	ordersModels "Encargalo.app-api.go/internal/orders/domain/models"
	"Encargalo.app-api.go/internal/riders/app"
	riderModels "Encargalo.app-api.go/internal/riders/domain/models"
	ridersPorts "Encargalo.app-api.go/internal/riders/domain/ports"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type orderLocationEvent struct {
	customerID uuid.UUID
	orderID    uuid.UUID
	lat, lng   float64
}

type recordingOrderNotifier struct {
	mu     sync.Mutex
	events []orderLocationEvent
}

func newRecordingOrderNotifier(t *testing.T) (*mocksriders.OrderCustomerNotifier, *recordingOrderNotifier) {
	t.Helper()
	rec := &recordingOrderNotifier{}
	m := mocksriders.NewOrderCustomerNotifier(t)
	m.On("NotifyRiderOrderLocation", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			rec.mu.Lock()
			defer rec.mu.Unlock()
			rec.events = append(rec.events, orderLocationEvent{args.Get(1).(uuid.UUID), args.Get(2).(uuid.UUID), args.Get(3).(float64), args.Get(4).(float64)})
		}).Return().Maybe()
	return m, rec
}

func (n *recordingOrderNotifier) take() []orderLocationEvent {
	n.mu.Lock()
	defer n.mu.Unlock()
	out := n.events
	n.events = nil
	return out
}

func newOrderLocationSvc(t *testing.T) (ridersPorts.RidersApp, *mocksriders.RidersRepo, *recordingOrderNotifier) {
	t.Helper()
	repo := mocksriders.NewRidersRepo(t)
	bcv := mockbcvports.NewBcvApp(t)
	bcv.On("GetRate", mock.Anything).Return(nil, errors.New("bcv down")).Maybe()
	repo.On("SearchShopByID", mock.Anything, mock.Anything).Return(&ordersModels.ShopOrder{}, nil).Maybe()
	svc := app.New(repo, newRidersStream(t), bcv, mockstrips.NewTripsRepo(t), mocksriders.NewTripEarningRecorder(t), nil, nil, nil, nil, newPresenceStore(t))
	mockNotifier, notifier := newRecordingOrderNotifier(t)
	setter, ok := svc.(ridersPorts.OrderCustomerNotifierSetter)
	require.True(t, ok, "appRidersManager debe implementar OrderCustomerNotifierSetter")
	setter.SetOrderCustomerNotifier(mockNotifier)
	return svc, repo, notifier
}

func acceptOrderFor(t *testing.T, svc ridersPorts.RidersApp, repo *mocksriders.RidersRepo, riderID, customerID uuid.UUID) uuid.UUID {
	t.Helper()
	orderID := uuid.New()
	order := &ordersModels.Order{ID: orderID, CustomerID: customerID, ShopID: uuid.New(), Status: string(ordersModels.StatusReady)}
	repo.On("SearchOrdersByID", mock.Anything, orderID).Return(order, nil)
	repo.On("AssignRider", mock.Anything, orderID, riderID).Return(nil)
	conn := mocksriders.NewRiderConn(t)
	conn.On("Send", mock.Anything).Return().Maybe()
	require.NoError(t, svc.AcceptOrder(context.Background(), riderID, orderID, conn))
	return orderID
}

func TestUpdateLocation_NotifiesCustomerOfEachAssignedOrder(t *testing.T) {
	svc, repo, notifier := newOrderLocationSvc(t)
	riderID := uuid.New()
	customerA, customerB := uuid.New(), uuid.New()
	orderA := acceptOrderFor(t, svc, repo, riderID, customerA)
	orderB := acceptOrderFor(t, svc, repo, riderID, customerB)

	require.NoError(t, svc.UpdateLocation(context.Background(), riderID, 10.48, -66.90))

	events := notifier.take()
	require.ElementsMatch(t, []orderLocationEvent{
		{customerA, orderA, 10.48, -66.90},
		{customerB, orderB, 10.48, -66.90},
	}, events)
}

func TestUpdateLocation_WithoutAssignedOrders_DoesNotNotify(t *testing.T) {
	svc, _, notifier := newOrderLocationSvc(t)

	require.NoError(t, svc.UpdateLocation(context.Background(), uuid.New(), 10.48, -66.90))

	require.Empty(t, notifier.take())
}

func TestUpdateLocation_OtherRiderOrdersAreNotNotified(t *testing.T) {
	svc, repo, notifier := newOrderLocationSvc(t)
	riderID, otherID := uuid.New(), uuid.New()
	acceptOrderFor(t, svc, repo, riderID, uuid.New())

	require.NoError(t, svc.UpdateLocation(context.Background(), otherID, 10.48, -66.90))

	require.Empty(t, notifier.take())
}

func TestAcceptOrder_SendsLastKnownLocationImmediately(t *testing.T) {
	svc, repo, notifier := newOrderLocationSvc(t)
	riderID, customerID := uuid.New(), uuid.New()
	require.NoError(t, svc.UpdateLocation(context.Background(), riderID, 10.50, -66.85))
	require.Empty(t, notifier.take())

	orderID := acceptOrderFor(t, svc, repo, riderID, customerID)

	require.Equal(t, []orderLocationEvent{{customerID, orderID, 10.50, -66.85}}, notifier.take())
}

func TestAcceptOrder_WithoutKnownLocation_DoesNotNotify(t *testing.T) {
	svc, repo, notifier := newOrderLocationSvc(t)

	acceptOrderFor(t, svc, repo, uuid.New(), uuid.New())

	require.Empty(t, notifier.take())
}

func TestUpdateLocation_StopsAfterOrderFinishes(t *testing.T) {
	for _, status := range []ordersModels.OrderStatus{ordersModels.StatusCompleted, ordersModels.StatusCancelled, ordersModels.StatusRejected} {
		t.Run(string(status), func(t *testing.T) {
			svc, repo, notifier := newOrderLocationSvc(t)
			riderID, customerID := uuid.New(), uuid.New()
			orderID := acceptOrderFor(t, svc, repo, riderID, customerID)

			asBroadcaster(t, svc).BroadcastOrderUpdate(context.Background(), ordersModels.Order{ID: orderID, CustomerID: customerID, Status: string(status)})
			require.NoError(t, svc.UpdateLocation(context.Background(), riderID, 10.48, -66.90))

			require.Empty(t, notifier.take())
		})
	}
}

func TestUpdateLocation_KeepsNotifyingWhileOrderIsOnTheWay(t *testing.T) {
	svc, repo, notifier := newOrderLocationSvc(t)
	riderID, customerID := uuid.New(), uuid.New()
	orderID := acceptOrderFor(t, svc, repo, riderID, customerID)

	asBroadcaster(t, svc).BroadcastOrderUpdate(context.Background(), ordersModels.Order{ID: orderID, CustomerID: customerID, Status: string(ordersModels.StatusOnTheWay)})
	require.NoError(t, svc.UpdateLocation(context.Background(), riderID, 10.48, -66.90))

	require.Equal(t, []orderLocationEvent{{customerID, orderID, 10.48, -66.90}}, notifier.take())
}

func TestRegister_RelinksActiveOrdersToLocationUpdates(t *testing.T) {
	svc, repo, notifier := newOrderLocationSvc(t)
	riderID, customerID := uuid.New(), uuid.New()
	active := ordersModels.Order{ID: uuid.New(), CustomerID: customerID, ShopID: uuid.New(), Status: string(ordersModels.StatusOnTheWay)}
	repo.On("SearchActiveOrdersByRiderID", mock.Anything, riderID).Return([]ordersModels.Order{active}, nil)
	repo.On("SearchAvailableOrders", mock.Anything).Return(nil, nil)
	conn := mocksriders.NewRiderConn(t)
	conn.On("Send", mock.Anything).Return().Maybe()
	require.NoError(t, svc.Register(context.Background(), riderID, conn, riderModels.RiderModeOrders))

	require.NoError(t, svc.UpdateLocation(context.Background(), riderID, 10.48, -66.90))

	require.Equal(t, []orderLocationEvent{{customerID, active.ID, 10.48, -66.90}}, notifier.take())
}
