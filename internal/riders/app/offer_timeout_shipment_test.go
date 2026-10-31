package app

import (
	"context"
	"testing"
	"time"

	mockbcvports "Encargalo.app-api.go/internal/mocks/bcv/domain/ports"
	mockpayments "Encargalo.app-api.go/internal/mocks/payments/domain/ports"
	mocksriders "Encargalo.app-api.go/internal/mocks/riders/domain/ports"
	"Encargalo.app-api.go/internal/pkg/money"
	shipmentsModels "Encargalo.app-api.go/internal/shipments/domain/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

// TestExpireShipmentIfStillSearching_RefundsBalance: igual que con trips, un
// vencimiento sin nadie que aceptara reembolsa lo que se debitó del balance.
func TestExpireShipmentIfStillSearching_RefundsBalance(t *testing.T) {
	repo := mocksriders.NewRidersRepo(t)
	stream := mocksriders.NewRidersStream(t)
	bcv := mockbcvports.NewBcvApp(t)
	shipmentsRepo := mocksriders.NewShipmentsStore(t)
	earnings := mocksriders.NewShipmentEarningRecorder(t)
	balance := mockpayments.NewCustomerBalanceCreditor(t)
	offerTimeouts := mocksriders.NewShipmentOfferTimeoutStore(t)
	mgr := New(repo, stream, bcv, mocksriders.NewTripsStore(t), mocksriders.NewTripEarningRecorder(t),
		nil, balance, nil, nil, nil, newOfferTimeoutStoreForTest(t), nil, nil,
		shipmentsRepo, offerTimeouts, earnings, nil).(*appRidersManager)

	shipmentID, customerID := uuid.New(), uuid.New()
	balancePaid := 32.0
	shipment := &shipmentsModels.Shipment{
		ID: shipmentID, CustomerID: customerID, Status: shipmentsModels.ShipmentStatusNoDriver,
		MethodPayment: shipmentsModels.MethodBalance, BalancePaidBs: &balancePaid,
	}

	shipmentsRepo.On("UpdateShipmentStatus", mock.Anything, shipmentID, shipmentsModels.ShipmentStatusSearching, shipmentsModels.ShipmentStatusNoDriver).Return(true, nil)
	shipmentsRepo.On("SaveShipmentCancelReason", mock.Anything, shipmentID, shipmentsModels.CancelReasonNoDriver).Return(nil)
	shipmentsRepo.On("GetShipmentByID", mock.Anything, shipmentID).Return(shipment, nil)
	balance.On("RefundBalancePayment", mock.Anything, customerID, shipmentID, shipmentsModels.ReferenceTypeShipment, balancePaid).Return(nil)
	stream.On("PublisherEvent", mock.Anything, mock.MatchedBy(func(e shipmentsModels.DataEventShipment) bool {
		return e.MessageType == shipmentsModels.EventShipmentNoDriver && e.ShipmentID == shipmentID.String()
	})).Return(nil)

	mgr.expireShipmentIfStillSearching(shipmentID)

	balance.AssertCalled(t, "RefundBalancePayment", mock.Anything, customerID, shipmentID, shipmentsModels.ReferenceTypeShipment, balancePaid)
}

// TestExpireShipmentIfStillSearching_CreditsUnservedPagoMovil: un Pago Móvil
// ya aprobado que nunca encontró rider se acredita a la cartera, no se reembolsa como balance.
func TestExpireShipmentIfStillSearching_CreditsUnservedPagoMovil(t *testing.T) {
	repo := mocksriders.NewRidersRepo(t)
	stream := mocksriders.NewRidersStream(t)
	bcv := mockbcvports.NewBcvApp(t)
	shipmentsRepo := mocksriders.NewShipmentsStore(t)
	earnings := mocksriders.NewShipmentEarningRecorder(t)
	balance := mockpayments.NewCustomerBalanceCreditor(t)
	offerTimeouts := mocksriders.NewShipmentOfferTimeoutStore(t)
	mgr := New(repo, stream, bcv, mocksriders.NewTripsStore(t), mocksriders.NewTripEarningRecorder(t),
		nil, balance, nil, nil, nil, newOfferTimeoutStoreForTest(t), nil, nil,
		shipmentsRepo, offerTimeouts, earnings, nil).(*appRidersManager)

	shipmentID, customerID, txID := uuid.New(), uuid.New(), uuid.New()
	fare, rate := 0.8, 40.0
	shipment := &shipmentsModels.Shipment{
		ID: shipmentID, CustomerID: customerID, Status: shipmentsModels.ShipmentStatusNoDriver,
		MethodPayment: shipmentsModels.MethodPagoMovil, Fare: &fare, BcvRate: &rate,
	}

	shipmentsRepo.On("UpdateShipmentStatus", mock.Anything, shipmentID, shipmentsModels.ShipmentStatusSearching, shipmentsModels.ShipmentStatusNoDriver).Return(true, nil)
	shipmentsRepo.On("SaveShipmentCancelReason", mock.Anything, shipmentID, shipmentsModels.CancelReasonNoDriver).Return(nil)
	shipmentsRepo.On("GetShipmentByID", mock.Anything, shipmentID).Return(shipment, nil)
	// money.RoundBs redondea a la decena más cercana: 0.8*40=32 -> 30.
	expectedBs := money.RoundBs(fare * rate)
	balance.On("CreditOrderRejected", mock.Anything, customerID, shipmentID, shipmentsModels.ReferenceTypeShipment, expectedBs, expectedBs, rate).
		Return(txID, time.Now(), nil)
	shipmentsRepo.On("SaveShipmentPayment", mock.Anything, shipmentID, mock.Anything).Return(nil)
	stream.On("PublisherEvent", mock.Anything, mock.Anything).Return(nil)

	mgr.expireShipmentIfStillSearching(shipmentID)

	balance.AssertCalled(t, "CreditOrderRejected", mock.Anything, customerID, shipmentID, shipmentsModels.ReferenceTypeShipment, expectedBs, expectedBs, rate)
	balance.AssertNotCalled(t, "RefundBalancePayment", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// TestOnShipmentOfferTimeout_IgnoresAlreadyAcceptedShipment: si el rider ya
// aceptó antes de que el worker reclame el vencimiento, no debe tocar nada.
func TestOnShipmentOfferTimeout_IgnoresAlreadyAcceptedShipment(t *testing.T) {
	repo := mocksriders.NewRidersRepo(t)
	stream := mocksriders.NewRidersStream(t)
	bcv := mockbcvports.NewBcvApp(t)
	shipmentsRepo := mocksriders.NewShipmentsStore(t)
	earnings := mocksriders.NewShipmentEarningRecorder(t)
	offerTimeouts := mocksriders.NewShipmentOfferTimeoutStore(t)
	mgr := New(repo, stream, bcv, mocksriders.NewTripsStore(t), mocksriders.NewTripEarningRecorder(t),
		nil, nil, nil, nil, nil, newOfferTimeoutStoreForTest(t), nil, nil,
		shipmentsRepo, offerTimeouts, earnings, nil).(*appRidersManager)

	shipmentID := uuid.New()
	shipmentsRepo.On("GetShipmentByID", mock.Anything, shipmentID).Return(&shipmentsModels.Shipment{ID: shipmentID, Status: shipmentsModels.ShipmentStatusAccepted}, nil)

	mgr.onShipmentOfferTimeout(context.Background(), shipmentID)

	shipmentsRepo.AssertNotCalled(t, "UpdateShipmentStatus", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// newOfferTimeoutStoreForTest es un stub nulo del TripOfferTimeoutStore (no
// relacionado con shipments): estos tests no ofertan viajes.
func newOfferTimeoutStoreForTest(t *testing.T) *mocksriders.TripOfferTimeoutStore {
	t.Helper()
	return mocksriders.NewTripOfferTimeoutStore(t)
}
