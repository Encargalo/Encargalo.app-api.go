package app_test

import (
	"context"
	"testing"

	bcvmodels "Encargalo.app-api.go/internal/bcv/domain/models"
	mockbcvports "Encargalo.app-api.go/internal/mocks/bcv/domain/ports"
	mocksriders "Encargalo.app-api.go/internal/mocks/riders/domain/ports"
	mocksshipments "Encargalo.app-api.go/internal/mocks/shipments/domain/ports"
	"Encargalo.app-api.go/internal/riders/app"
	riderModels "Encargalo.app-api.go/internal/riders/domain/models"
	ridersPorts "Encargalo.app-api.go/internal/riders/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	shipmentsModels "Encargalo.app-api.go/internal/shipments/domain/models"
	shipmentsPorts "Encargalo.app-api.go/internal/shipments/domain/ports"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// shipmentsMgr arma un manager con los colaboradores de shipments mockeados,
// igual que ordersMgr para pedidos. El resto (trips/balance/passengers/vehicles)
// queda en nil: ninguno de estos tests los necesita.
func shipmentsMgr(t *testing.T) (ridersPorts.RidersApp, *mocksriders.RidersRepo, *mockbcvports.BcvApp, *mocksriders.ShipmentsStore, *mocksriders.ShipmentOfferTimeoutStore, *mocksriders.ShipmentEarningRecorder, *mocksshipments.CustomerShipmentNotifier) {
	t.Helper()
	repo := mocksriders.NewRidersRepo(t)
	stream := newRidersStream(t)
	bcv := mockbcvports.NewBcvApp(t)
	shipmentsRepo := mocksriders.NewShipmentsStore(t)
	offerTimeouts := mocksriders.NewShipmentOfferTimeoutStore(t)
	earnings := mocksriders.NewShipmentEarningRecorder(t)
	customers := mocksshipments.NewCustomerShipmentNotifier(t)
	svc := app.New(repo, stream, bcv, mocksriders.NewTripsStore(t), mocksriders.NewTripEarningRecorder(t),
		nil, nil, nil, nil, newPresenceStore(t), newOfferTimeoutStore(t), nil, nil,
		shipmentsRepo, offerTimeouts, earnings, customers)
	return svc, repo, bcv, shipmentsRepo, offerTimeouts, earnings, customers
}

// connectShipmentsRider conecta un rider en modo orders (el único pool que
// recibe envíos) sin nada asignado al inicio, y devuelve lo que el servidor le mande.
func connectShipmentsRider(t *testing.T, svc ridersPorts.RidersApp, repo *mocksriders.RidersRepo, shipmentsRepo *mocksriders.ShipmentsStore) (uuid.UUID, chan []byte) {
	t.Helper()
	id := uuid.New()
	repo.On("SearchActiveOrdersByRiderID", mock.Anything, id).Return(nil, errcustom.ErrNotFound).Once()
	repo.On("SearchAvailableOrders", mock.Anything).Return(nil, errcustom.ErrNotFound).Once()
	shipmentsRepo.On("ListActiveShipmentsByRider", mock.Anything, id).Return(nil, errcustom.ErrNotFound).Once()

	sent := make(chan []byte, 8)
	conn := mocksriders.NewRiderConn(t)
	conn.On("Send", mock.Anything).Run(func(args mock.Arguments) {
		sent <- args.Get(0).([]byte)
	}).Return().Maybe()

	require.NoError(t, svc.Register(context.Background(), id, conn, riderModels.RiderModeOrders))
	return id, sent
}

func asShipmentMatcher(t *testing.T, svc ridersPorts.RidersApp) shipmentsPorts.ShipmentRiderMatcher {
	t.Helper()
	m, ok := svc.(shipmentsPorts.ShipmentRiderMatcher)
	require.True(t, ok, "appRidersManager debe implementar shipmentsPorts.ShipmentRiderMatcher")
	return m
}

func asShipmentNotifier(t *testing.T, svc ridersPorts.RidersApp) shipmentsPorts.ShipmentRiderNotifier {
	t.Helper()
	n, ok := svc.(shipmentsPorts.ShipmentRiderNotifier)
	require.True(t, ok, "appRidersManager debe implementar shipmentsPorts.ShipmentRiderNotifier")
	return n
}

// --- OfferShipment: broadcast abierto, sin filtro de vehículo ---

func TestOfferShipment_BroadcastsOnlyToOrdersModeRiders(t *testing.T) {
	svc, repo, _, shipmentsRepo, offerTimeouts, _, _ := shipmentsMgr(t)
	_, ordersSent := connectShipmentsRider(t, svc, repo, shipmentsRepo)

	tripsRiderID := uuid.New()
	repo.On("SearchActiveOrdersByRiderID", mock.Anything, tripsRiderID).Return(nil, errcustom.ErrNotFound).Once()
	repo.On("SearchAvailableOrders", mock.Anything).Return(nil, errcustom.ErrNotFound).Once()
	shipmentsRepo.On("ListActiveShipmentsByRider", mock.Anything, tripsRiderID).Return(nil, errcustom.ErrNotFound).Once()
	tripsConn := mocksriders.NewRiderConn(t)
	require.NoError(t, svc.Register(context.Background(), tripsRiderID, tripsConn, riderModels.RiderModeTrips))

	shipmentID := uuid.New()
	offerTimeouts.On("Schedule", mock.Anything, shipmentID, mock.AnythingOfType("time.Time")).Return(nil)

	asShipmentMatcher(t, svc).OfferShipment(context.Background(), shipmentsModels.Shipment{ID: shipmentID, Status: shipmentsModels.ShipmentStatusSearching})

	require.Equal(t, "new_shipment", messageType(t, <-ordersSent))
	tripsConn.AssertNotCalled(t, "Send", mock.Anything)
}

func TestOfferShipment_SkipsPausedRider(t *testing.T) {
	svc, repo, _, shipmentsRepo, offerTimeouts, _, _ := shipmentsMgr(t)
	riderID, sent := connectShipmentsRider(t, svc, repo, shipmentsRepo)
	require.NoError(t, svc.SetAvailability(context.Background(), riderID, false))

	shipmentID := uuid.New()
	offerTimeouts.On("Schedule", mock.Anything, shipmentID, mock.AnythingOfType("time.Time")).Return(nil)
	asShipmentMatcher(t, svc).OfferShipment(context.Background(), shipmentsModels.Shipment{ID: shipmentID, Status: shipmentsModels.ShipmentStatusSearching})

	select {
	case msg := <-sent:
		t.Fatalf("rider pausado no debería recibir new_shipment: %s", msg)
	default:
	}
}

// --- AcceptShipment ---

func TestAcceptShipment_CASLossSendsShipmentTaken(t *testing.T) {
	svc, repo, _, shipmentsRepo, offerTimeouts, _, _ := shipmentsMgr(t)
	riderID, _ := connectShipmentsRider(t, svc, repo, shipmentsRepo)
	shipmentID := uuid.New()

	shipmentsRepo.On("GetShipmentByID", mock.Anything, shipmentID).Return(&shipmentsModels.Shipment{ID: shipmentID, Status: shipmentsModels.ShipmentStatusSearching}, nil)
	shipmentsRepo.On("AssignShipmentRider", mock.Anything, shipmentID, riderID).Return(false, nil)

	conn := mocksriders.NewRiderConn(t)
	var sent []byte
	conn.On("Send", mock.Anything).Run(func(args mock.Arguments) { sent = args.Get(0).([]byte) }).Return()

	err := svc.AcceptShipment(context.Background(), riderID, shipmentID, conn)

	require.NoError(t, err)
	require.Equal(t, "shipment_taken", messageType(t, sent))
	offerTimeouts.AssertNotCalled(t, "Cancel", mock.Anything, mock.Anything)
}

func TestAcceptShipment_WrongStatusRejected(t *testing.T) {
	svc, repo, _, shipmentsRepo, _, _, _ := shipmentsMgr(t)
	riderID, _ := connectShipmentsRider(t, svc, repo, shipmentsRepo)
	shipmentID := uuid.New()
	shipmentsRepo.On("GetShipmentByID", mock.Anything, shipmentID).Return(&shipmentsModels.Shipment{ID: shipmentID, Status: shipmentsModels.ShipmentStatusAccepted}, nil)

	err := svc.AcceptShipment(context.Background(), riderID, shipmentID, mocksriders.NewRiderConn(t))

	require.ErrorIs(t, err, errcustom.ErrInvalidTransition)
	shipmentsRepo.AssertNotCalled(t, "AssignShipmentRider", mock.Anything, mock.Anything, mock.Anything)
}

func TestAcceptShipment_Success(t *testing.T) {
	svc, repo, _, shipmentsRepo, offerTimeouts, _, customers := shipmentsMgr(t)
	riderID, _ := connectShipmentsRider(t, svc, repo, shipmentsRepo)
	customerID, shipmentID := uuid.New(), uuid.New()

	shipmentsRepo.On("GetShipmentByID", mock.Anything, shipmentID).Return(&shipmentsModels.Shipment{ID: shipmentID, CustomerID: customerID, Status: shipmentsModels.ShipmentStatusSearching}, nil).Once()
	shipmentsRepo.On("AssignShipmentRider", mock.Anything, shipmentID, riderID).Return(true, nil)
	offerTimeouts.On("Cancel", mock.Anything, shipmentID).Return(nil)
	accepted := &shipmentsModels.Shipment{
		ID: shipmentID, CustomerID: customerID, RiderID: &riderID, Status: shipmentsModels.ShipmentStatusAccepted,
		PickupCode: "1234", DropoffCode: "5678",
	}
	shipmentsRepo.On("GetShipmentByID", mock.Anything, shipmentID).Return(accepted, nil)
	customers.On("NotifyShipment", mock.Anything, mock.MatchedBy(func(s shipmentsModels.Shipment) bool {
		return s.ID == shipmentID && s.Status == shipmentsModels.ShipmentStatusAccepted
	})).Return()

	conn := mocksriders.NewRiderConn(t)
	var sent []byte
	conn.On("Send", mock.Anything).Run(func(args mock.Arguments) { sent = args.Get(0).([]byte) }).Return()

	err := svc.AcceptShipment(context.Background(), riderID, shipmentID, conn)

	require.NoError(t, err)
	require.Equal(t, "shipment_accepted", messageType(t, sent))
	// El WS del rider nunca lleva los códigos de verificación (ToDTO, no ToCustomerDTO).
	require.NotContains(t, string(sent), "pickup_code")
	require.NotContains(t, string(sent), "dropoff_code")
}

// --- AdvanceShipment ---

func TestAdvanceShipment_WrongRiderRejected(t *testing.T) {
	svc, repo, _, shipmentsRepo, _, _, _ := shipmentsMgr(t)
	riderID, _ := connectShipmentsRider(t, svc, repo, shipmentsRepo)
	otherRiderID, shipmentID := uuid.New(), uuid.New()
	shipmentsRepo.On("GetShipmentByID", mock.Anything, shipmentID).Return(&shipmentsModels.Shipment{ID: shipmentID, RiderID: &otherRiderID, Status: shipmentsModels.ShipmentStatusAccepted}, nil)

	err := svc.AdvanceShipment(context.Background(), riderID, shipmentID, "1234", mocksriders.NewRiderConn(t))

	require.ErrorIs(t, err, errcustom.ErrInvalidTransition)
	shipmentsRepo.AssertNotCalled(t, "AdvanceShipmentStatus", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestAdvanceShipment_WrongCodeReturnsErrInvalidCode(t *testing.T) {
	svc, repo, _, shipmentsRepo, _, _, _ := shipmentsMgr(t)
	riderID, _ := connectShipmentsRider(t, svc, repo, shipmentsRepo)
	shipmentID := uuid.New()
	shipmentsRepo.On("GetShipmentByID", mock.Anything, shipmentID).Return(&shipmentsModels.Shipment{ID: shipmentID, RiderID: &riderID, Status: shipmentsModels.ShipmentStatusAccepted}, nil)
	shipmentsRepo.On("AdvanceShipmentStatus", mock.Anything, shipmentID, riderID, "0000").Return(false, nil)

	err := svc.AdvanceShipment(context.Background(), riderID, shipmentID, "0000", mocksriders.NewRiderConn(t))

	require.ErrorIs(t, err, errcustom.ErrInvalidCode)
}

func TestAdvanceShipment_PickupCodeTransitionsToInTransit(t *testing.T) {
	svc, repo, _, shipmentsRepo, _, _, customers := shipmentsMgr(t)
	riderID, _ := connectShipmentsRider(t, svc, repo, shipmentsRepo)
	shipmentID := uuid.New()
	shipmentsRepo.On("GetShipmentByID", mock.Anything, shipmentID).Return(&shipmentsModels.Shipment{ID: shipmentID, RiderID: &riderID, Status: shipmentsModels.ShipmentStatusAccepted}, nil).Once()
	shipmentsRepo.On("AdvanceShipmentStatus", mock.Anything, shipmentID, riderID, "1234").Return(true, nil)
	inTransit := &shipmentsModels.Shipment{ID: shipmentID, RiderID: &riderID, Status: shipmentsModels.ShipmentStatusInTransit}
	shipmentsRepo.On("GetShipmentByID", mock.Anything, shipmentID).Return(inTransit, nil)
	customers.On("NotifyShipment", mock.Anything, mock.Anything).Return()

	conn := mocksriders.NewRiderConn(t)
	var sent []byte
	conn.On("Send", mock.Anything).Run(func(args mock.Arguments) { sent = args.Get(0).([]byte) }).Return()

	err := svc.AdvanceShipment(context.Background(), riderID, shipmentID, "1234", conn)

	require.NoError(t, err)
	require.Equal(t, "shipment_status", messageType(t, sent))
}

func TestAdvanceShipment_InTransitWithEmptyCodeReturnsErrInvalidCode(t *testing.T) {
	svc, repo, _, shipmentsRepo, _, _, _ := shipmentsMgr(t)
	riderID, _ := connectShipmentsRider(t, svc, repo, shipmentsRepo)
	shipmentID := uuid.New()
	fare, rate := 0.8, 40.0
	shipmentsRepo.On("GetShipmentByID", mock.Anything, shipmentID).Return(&shipmentsModels.Shipment{
		ID: shipmentID, RiderID: &riderID, Status: shipmentsModels.ShipmentStatusInTransit, Fare: &fare, BcvRate: &rate,
	}, nil)

	err := svc.AdvanceShipment(context.Background(), riderID, shipmentID, "   ", mocksriders.NewRiderConn(t))

	require.ErrorIs(t, err, errcustom.ErrInvalidCode)
	shipmentsRepo.AssertNotCalled(t, "AdvanceShipmentStatus", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// TestAdvanceShipment_DropoffCodeDeliversAndRecordsEarning: la ganancia solo
// se liquida en la transición final (in_transit -> delivered), nunca en la de retiro.
func TestAdvanceShipment_DropoffCodeDeliversAndRecordsEarning(t *testing.T) {
	svc, repo, bcv, shipmentsRepo, _, earnings, customers := shipmentsMgr(t)
	riderID, _ := connectShipmentsRider(t, svc, repo, shipmentsRepo)
	shipmentID := uuid.New()
	fare, rate := 0.8, 40.0
	bcv.On("GetRate", mock.Anything).Return(&bcvmodels.ExchangeRate{Rate: rate}, nil)

	shipmentsRepo.On("GetShipmentByID", mock.Anything, shipmentID).Return(&shipmentsModels.Shipment{
		ID: shipmentID, RiderID: &riderID, Status: shipmentsModels.ShipmentStatusInTransit, Fare: &fare, BcvRate: &rate,
	}, nil).Once()
	shipmentsRepo.On("AdvanceShipmentStatus", mock.Anything, shipmentID, riderID, "5678").Return(true, nil)
	delivered := &shipmentsModels.Shipment{ID: shipmentID, RiderID: &riderID, Status: shipmentsModels.ShipmentStatusDelivered, Fare: &fare, BcvRate: &rate}
	shipmentsRepo.On("GetShipmentByID", mock.Anything, shipmentID).Return(delivered, nil)
	customers.On("NotifyShipment", mock.Anything, mock.Anything).Return()
	earnings.On("RecordShipmentEarning", mock.Anything, riderID, shipmentID, fare, rate).Return(nil)

	conn := mocksriders.NewRiderConn(t)
	var sent []byte
	conn.On("Send", mock.Anything).Run(func(args mock.Arguments) { sent = args.Get(0).([]byte) }).Return()

	err := svc.AdvanceShipment(context.Background(), riderID, shipmentID, "5678", conn)

	require.NoError(t, err)
	require.Equal(t, "shipment_status", messageType(t, sent))
	earnings.AssertCalled(t, "RecordShipmentEarning", mock.Anything, riderID, shipmentID, fare, rate)
}

// --- CancelAcceptedShipment ---

func TestCancelAcceptedShipment_WrongStatusRejected(t *testing.T) {
	svc, repo, _, shipmentsRepo, _, _, _ := shipmentsMgr(t)
	riderID, _ := connectShipmentsRider(t, svc, repo, shipmentsRepo)
	shipmentID := uuid.New()
	shipmentsRepo.On("GetShipmentByID", mock.Anything, shipmentID).Return(&shipmentsModels.Shipment{ID: shipmentID, RiderID: &riderID, Status: shipmentsModels.ShipmentStatusInTransit}, nil)

	err := svc.CancelAcceptedShipment(context.Background(), riderID, shipmentID, mocksriders.NewRiderConn(t))

	require.ErrorIs(t, err, errcustom.ErrInvalidTransition)
	shipmentsRepo.AssertNotCalled(t, "ReleaseShipmentRider", mock.Anything, mock.Anything, mock.Anything)
}

// TestCancelAcceptedShipment_ReleasesAndReoffersToPool: a diferencia de trips,
// no hay lista de exclusión — el re-broadcast llega a todo el pool, incluido
// el rider que acaba de cancelar.
func TestCancelAcceptedShipment_ReleasesAndReoffersToPool(t *testing.T) {
	svc, repo, _, shipmentsRepo, offerTimeouts, _, customers := shipmentsMgr(t)
	riderID, riderSent := connectShipmentsRider(t, svc, repo, shipmentsRepo)
	_, otherSent := connectShipmentsRider(t, svc, repo, shipmentsRepo)
	shipmentID := uuid.New()

	shipmentsRepo.On("GetShipmentByID", mock.Anything, shipmentID).Return(&shipmentsModels.Shipment{ID: shipmentID, RiderID: &riderID, Status: shipmentsModels.ShipmentStatusAccepted}, nil).Once()
	shipmentsRepo.On("ReleaseShipmentRider", mock.Anything, shipmentID, riderID).Return(true, nil)
	searching := &shipmentsModels.Shipment{ID: shipmentID, Status: shipmentsModels.ShipmentStatusSearching}
	shipmentsRepo.On("GetShipmentByID", mock.Anything, shipmentID).Return(searching, nil)
	customers.On("NotifyShipment", mock.Anything, mock.Anything).Return()
	offerTimeouts.On("Schedule", mock.Anything, shipmentID, mock.AnythingOfType("time.Time")).Return(nil)

	cancelConn := mocksriders.NewRiderConn(t)
	var cancelSent []byte
	cancelConn.On("Send", mock.Anything).Run(func(args mock.Arguments) { cancelSent = args.Get(0).([]byte) }).Return()

	err := svc.CancelAcceptedShipment(context.Background(), riderID, shipmentID, cancelConn)

	require.NoError(t, err)
	require.Equal(t, "shipment_status", messageType(t, cancelSent))
	require.Equal(t, "new_shipment", messageType(t, <-riderSent))
	require.Equal(t, "new_shipment", messageType(t, <-otherSent))
}

// --- NotifyShipmentCancelled ---

func TestNotifyShipmentCancelled_NotifiesAssignedRiderOnly(t *testing.T) {
	svc, repo, _, shipmentsRepo, offerTimeouts, _, _ := shipmentsMgr(t)
	riderID, riderSent := connectShipmentsRider(t, svc, repo, shipmentsRepo)
	_, otherSent := connectShipmentsRider(t, svc, repo, shipmentsRepo)
	shipmentID := uuid.New()
	offerTimeouts.On("Cancel", mock.Anything, shipmentID).Return(nil)

	asShipmentNotifier(t, svc).NotifyShipmentCancelled(context.Background(), shipmentsModels.Shipment{ID: shipmentID, RiderID: &riderID})

	require.Equal(t, "shipment_cancelled", messageType(t, <-riderSent))
	select {
	case msg := <-otherSent:
		t.Fatalf("el rider no asignado no debería recibir nada: %s", msg)
	default:
	}
}

func TestNotifyShipmentCancelled_NoRiderAssignedStillCancelsTimeout(t *testing.T) {
	svc, _, _, _, offerTimeouts, _, _ := shipmentsMgr(t)
	shipmentID := uuid.New()
	offerTimeouts.On("Cancel", mock.Anything, shipmentID).Return(nil)

	asShipmentNotifier(t, svc).NotifyShipmentCancelled(context.Background(), shipmentsModels.Shipment{ID: shipmentID})

	offerTimeouts.AssertCalled(t, "Cancel", mock.Anything, shipmentID)
}
