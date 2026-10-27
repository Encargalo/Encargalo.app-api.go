package app_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	bcvmodels "Encargalo.app-api.go/internal/bcv/domain/models"
	ordermodels "Encargalo.app-api.go/internal/orders/domain/models"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestUpdateOrderStatus_Success(t *testing.T) {
	svc, m := newOrderAppFixture(t)
	orderID, shopID, customerID := uuid.New(), uuid.New(), uuid.New()
	order := &ordermodels.Order{ID: orderID, ShopID: shopID, CustomerID: customerID, Status: string(ordermodels.StatusAccepted), MethodPayment: "Cash"}

	m.repo.On("SearchOrdersByID", mock.Anything, orderID).Return(order, nil)
	m.repo.On("UpdateOrderStatus", mock.Anything, orderID, string(ordermodels.StatusReady)).Return(nil)
	m.bcv.On("GetRate", mock.Anything).Return(&bcvmodels.ExchangeRate{Rate: 40}, nil)
	m.riderMgr.On("BroadcastOrderUpdate", mock.Anything, mock.Anything).Return()

	out, err := svc.UpdateOrderStatus(context.Background(), orderID, "Ready", shopID, true, shopID)

	require.NoError(t, err)
	assert.Equal(t, string(ordermodels.StatusReady), out.Status)
}

func TestUpdateOrderStatus_WrongShopUnauthorized(t *testing.T) {
	svc, m := newOrderAppFixture(t)
	orderID, shopID, otherShopID := uuid.New(), uuid.New(), uuid.New()
	order := &ordermodels.Order{ID: orderID, ShopID: shopID}
	m.repo.On("SearchOrdersByID", mock.Anything, orderID).Return(order, nil)

	_, err := svc.UpdateOrderStatus(context.Background(), orderID, "Ready", otherShopID, true, otherShopID)

	assert.ErrorIs(t, err, errcustom.ErrUnautorized)
}

func TestUpdateOrderStatus_CustomerCannotUpdateDirectly(t *testing.T) {
	svc, m := newOrderAppFixture(t)
	orderID, customerID := uuid.New(), uuid.New()
	order := &ordermodels.Order{ID: orderID, CustomerID: customerID}
	m.repo.On("SearchOrdersByID", mock.Anything, orderID).Return(order, nil)

	_, err := svc.UpdateOrderStatus(context.Background(), orderID, "Accepted", customerID, false, uuid.Nil)

	assert.ErrorIs(t, err, errcustom.ErrUnautorized)
}

func TestUpdateOrderStatus_InvalidStatus(t *testing.T) {
	svc, m := newOrderAppFixture(t)
	orderID, shopID := uuid.New(), uuid.New()
	order := &ordermodels.Order{ID: orderID, ShopID: shopID}
	m.repo.On("SearchOrdersByID", mock.Anything, orderID).Return(order, nil)

	_, err := svc.UpdateOrderStatus(context.Background(), orderID, "Bogus", shopID, true, shopID)

	assert.ErrorIs(t, err, errcustom.ErrInvalidStatus)
}

func TestUpdateOrderStatus_ShopCannotMarkCompleted(t *testing.T) {
	svc, m := newOrderAppFixture(t)
	orderID, shopID := uuid.New(), uuid.New()
	order := &ordermodels.Order{ID: orderID, ShopID: shopID}
	m.repo.On("SearchOrdersByID", mock.Anything, orderID).Return(order, nil)

	_, err := svc.UpdateOrderStatus(context.Background(), orderID, "Completed", shopID, true, shopID)

	assert.ErrorIs(t, err, errcustom.ErrUnautorized)
}

// TestUpdateOrderStatus_RejectedRefundsCustomer cubre el camino de negocio más
// sensible de UpdateOrderStatus: al rechazar una orden pagada por PagoMovil,
// refundCustomerOnShopReject debe acreditar el total al balance del cliente.
func TestUpdateOrderStatus_RejectedRefundsCustomer(t *testing.T) {
	svc, m := newOrderAppFixture(t)
	orderID, shopID, customerID := uuid.New(), uuid.New(), uuid.New()

	details, err := json.Marshal(ordermodels.Details{
		PaymentDetails: &ordermodels.PaymentDetails{BcvRate: 40, PagoMovilAmountBs: 880},
	})
	require.NoError(t, err)
	order := &ordermodels.Order{
		ID: orderID, ShopID: shopID, CustomerID: customerID,
		Status: string(ordermodels.StatusAccepted), MethodPayment: "PagoMovil",
		TotalPrice: 22, Details: details,
	}

	m.repo.On("SearchOrdersByID", mock.Anything, orderID).Return(order, nil)
	m.repo.On("UpdateOrderStatus", mock.Anything, orderID, string(ordermodels.StatusRejected)).Return(nil)
	m.bcv.On("GetRate", mock.Anything).Return(&bcvmodels.ExchangeRate{Rate: 40}, nil)
	m.customerBalance.On("GetBalancePaymentDebitForReference", mock.Anything, customerID, orderID, "order").Return(0.0, nil)
	m.customerBalance.On("CreditOrderRejected", mock.Anything, customerID, orderID, "order", 880.0, 880.0, 40.0).Return(uuid.New(), time.Now(), nil)
	m.repo.On("UpdateOrderDetails", mock.Anything, orderID, mock.Anything).Return(nil)
	m.riderMgr.On("BroadcastOrderUpdate", mock.Anything, mock.Anything).Return()
	m.shops.On("TryCloseOpenBatch", mock.Anything, shopID).Return(nil)
	m.stream.On("PublisherEvent", mock.Anything, mock.MatchedBy(func(e ordermodels.DataEventOrderStatusChanged) bool {
		return e.OrderID == orderID.String() && e.Status == string(ordermodels.StatusRejected)
	})).Return(nil)

	out, err := svc.UpdateOrderStatus(context.Background(), orderID, "Rejected", shopID, true, shopID)

	require.NoError(t, err)
	assert.Equal(t, string(ordermodels.StatusRejected), out.Status)
	m.stream.AssertCalled(t, "PublisherEvent", mock.Anything, mock.MatchedBy(func(e ordermodels.DataEventOrderStatusChanged) bool {
		return e.OrderID == orderID.String() && e.Status == string(ordermodels.StatusRejected)
	}))
}

func TestAutoRejectStalePending_RejectsAndRefunds(t *testing.T) {
	svc, m := newOrderAppFixture(t)
	orderID, shopID, customerID := uuid.New(), uuid.New(), uuid.New()
	stale := ordermodels.Order{ID: orderID, ShopID: shopID, CustomerID: customerID, MethodPayment: "Cash", Status: string(ordermodels.StatusPending)}

	m.repo.On("ListStalePending", mock.Anything, mock.Anything).Return([]ordermodels.Order{stale}, nil)
	m.repo.On("RejectIfPending", mock.Anything, orderID).Return(true, nil)
	m.shops.On("TryCloseOpenBatch", mock.Anything, shopID).Return(nil)
	m.stream.On("PublisherEvent", mock.Anything, mock.Anything).Return(nil)

	rejected, err := svc.AutoRejectStalePending(context.Background())

	require.NoError(t, err)
	require.Len(t, rejected, 1)
	assert.Equal(t, string(ordermodels.StatusRejected), rejected[0].Status)
}

func TestAutoRejectStalePending_NoneStale(t *testing.T) {
	svc, m := newOrderAppFixture(t)
	m.repo.On("ListStalePending", mock.Anything, mock.Anything).Return([]ordermodels.Order{}, nil)

	rejected, err := svc.AutoRejectStalePending(context.Background())

	require.NoError(t, err)
	assert.Empty(t, rejected)
}

func TestNotifyPendingUnassigned_PublishesAlertAtMinuteFour(t *testing.T) {
	svc, m := newOrderAppFixture(t)
	orderID, shopID := uuid.New(), uuid.New()
	pendingAt := time.Now().Add(-5 * time.Minute)
	pending := ordermodels.Order{ID: orderID, ShopID: shopID, PendingAt: &pendingAt, PendingAlertMinute: 0}

	m.repo.On("ListPendingUnassignedForAlert", mock.Anything, mock.Anything).Return([]ordermodels.Order{pending}, nil)
	m.repo.On("MarkPendingAlertMinute", mock.Anything, orderID, 4).Return(true, nil)
	m.stream.On("PublisherEvent", mock.Anything, mock.Anything).Return(nil)

	err := svc.NotifyPendingUnassigned(context.Background())

	require.NoError(t, err)
}

func TestNotifyPendingUnassigned_SkipsAlreadyAlerted(t *testing.T) {
	svc, m := newOrderAppFixture(t)
	orderID, shopID := uuid.New(), uuid.New()
	pendingAt := time.Now().Add(-5 * time.Minute)
	// PendingAlertMinute ya en 4: no debe volver a marcar/publicar para ese minuto.
	pending := ordermodels.Order{ID: orderID, ShopID: shopID, PendingAt: &pendingAt, PendingAlertMinute: 4}

	m.repo.On("ListPendingUnassignedForAlert", mock.Anything, mock.Anything).Return([]ordermodels.Order{pending}, nil)

	err := svc.NotifyPendingUnassigned(context.Background())

	require.NoError(t, err)
}

func TestConfirmPickup_Success(t *testing.T) {
	svc, m := newOrderAppFixture(t)
	orderID, shopID := uuid.New(), uuid.New()
	code := "1234"
	order := &ordermodels.Order{ID: orderID, ShopID: shopID, Status: string(ordermodels.StatusReady), PickupCode: &code}

	m.repo.On("SearchOrdersByID", mock.Anything, orderID).Return(order, nil)
	m.repo.On("MarkPickupCodeUsed", mock.Anything, orderID).Return(nil)
	m.repo.On("SaveDeliveryCode", mock.Anything, orderID, mock.AnythingOfType("string")).Return(nil)
	m.repo.On("UpdateOrderStatus", mock.Anything, orderID, string(ordermodels.StatusOnTheWay)).Return(nil)
	m.bcv.On("GetRate", mock.Anything).Return(&bcvmodels.ExchangeRate{Rate: 40}, nil)
	m.riderMgr.On("BroadcastOrderUpdate", mock.Anything, mock.Anything).Return()

	out, err := svc.ConfirmPickup(context.Background(), orderID, code, shopID)

	require.NoError(t, err)
	assert.Equal(t, string(ordermodels.StatusOnTheWay), out.Status)
	assert.NotNil(t, out.DeliveryCode)
}

func TestConfirmPickup_WrongShop(t *testing.T) {
	svc, m := newOrderAppFixture(t)
	orderID, shopID, otherShopID := uuid.New(), uuid.New(), uuid.New()
	order := &ordermodels.Order{ID: orderID, ShopID: shopID}
	m.repo.On("SearchOrdersByID", mock.Anything, orderID).Return(order, nil)

	_, err := svc.ConfirmPickup(context.Background(), orderID, "1234", otherShopID)

	assert.ErrorIs(t, err, errcustom.ErrUnautorized)
}

func TestConfirmPickup_InvalidCode(t *testing.T) {
	svc, m := newOrderAppFixture(t)
	orderID, shopID := uuid.New(), uuid.New()
	code := "1234"
	order := &ordermodels.Order{ID: orderID, ShopID: shopID, Status: string(ordermodels.StatusReady), PickupCode: &code}
	m.repo.On("SearchOrdersByID", mock.Anything, orderID).Return(order, nil)

	_, err := svc.ConfirmPickup(context.Background(), orderID, "0000", shopID)

	assert.ErrorIs(t, err, errcustom.ErrInvalidCode)
}

func TestConfirmPickup_CodeAlreadyUsed(t *testing.T) {
	svc, m := newOrderAppFixture(t)
	orderID, shopID := uuid.New(), uuid.New()
	code := "1234"
	usedAt := time.Now()
	order := &ordermodels.Order{ID: orderID, ShopID: shopID, Status: string(ordermodels.StatusReady), PickupCode: &code, PickupCodeUsedAt: &usedAt}
	m.repo.On("SearchOrdersByID", mock.Anything, orderID).Return(order, nil)

	_, err := svc.ConfirmPickup(context.Background(), orderID, code, shopID)

	assert.ErrorIs(t, err, errcustom.ErrCodeAlreadyUsed)
}

func TestConfirmPickup_InvalidTransition(t *testing.T) {
	svc, m := newOrderAppFixture(t)
	orderID, shopID := uuid.New(), uuid.New()
	order := &ordermodels.Order{ID: orderID, ShopID: shopID, Status: string(ordermodels.StatusPending)}
	m.repo.On("SearchOrdersByID", mock.Anything, orderID).Return(order, nil)

	_, err := svc.ConfirmPickup(context.Background(), orderID, "1234", shopID)

	assert.ErrorIs(t, err, errcustom.ErrInvalidTransition)
}

func TestConfirmDelivery_Success(t *testing.T) {
	svc, m := newOrderAppFixture(t)
	orderID, riderID, shopID := uuid.New(), uuid.New(), uuid.New()
	code := "5678"
	order := &ordermodels.Order{
		ID: orderID, ShopID: shopID, RiderID: &riderID, Status: string(ordermodels.StatusOnTheWay),
		DeliveryCode: &code, MethodPayment: "Cash", TotalPrice: 20, DeliveryFee: 2,
	}

	m.repo.On("SearchOrdersByID", mock.Anything, orderID).Return(order, nil)
	m.repo.On("MarkDeliveryCodeUsed", mock.Anything, orderID).Return(nil)
	m.repo.On("UpdateOrderStatus", mock.Anything, orderID, string(ordermodels.StatusCompleted)).Return(nil)
	m.bcv.On("GetRate", mock.Anything).Return(&bcvmodels.ExchangeRate{Rate: 40}, nil)
	m.finance.On("RecordRideEarning", mock.Anything, riderID, orderID, 2.0, 20.0, 40.0, "Cash").Return(nil)
	m.shopCommission.On("RecordSaleCommission", mock.Anything, shopID, orderID, 20.0, 2.0, "Cash", false, false, 0.0).Return(nil)
	m.riderMgr.On("BroadcastOrderUpdate", mock.Anything, mock.Anything).Return()
	m.shops.On("TryCloseOpenBatch", mock.Anything, shopID).Return(nil)
	m.stream.On("PublisherEvent", mock.Anything, mock.MatchedBy(func(e ordermodels.DataEventOrderStatusChanged) bool {
		return e.OrderID == orderID.String() && e.Status == string(ordermodels.StatusCompleted)
	})).Return(nil)

	out, err := svc.ConfirmDelivery(context.Background(), orderID, code, riderID)

	require.NoError(t, err)
	assert.Equal(t, string(ordermodels.StatusCompleted), out.Status)
}

func TestConfirmDelivery_WrongRider(t *testing.T) {
	svc, m := newOrderAppFixture(t)
	orderID, riderID, otherRiderID := uuid.New(), uuid.New(), uuid.New()
	order := &ordermodels.Order{ID: orderID, RiderID: &riderID}
	m.repo.On("SearchOrdersByID", mock.Anything, orderID).Return(order, nil)

	_, err := svc.ConfirmDelivery(context.Background(), orderID, "5678", otherRiderID)

	assert.ErrorIs(t, err, errcustom.ErrUnautorized)
}

func TestConfirmDelivery_InvalidCode(t *testing.T) {
	svc, m := newOrderAppFixture(t)
	orderID, riderID := uuid.New(), uuid.New()
	code := "5678"
	order := &ordermodels.Order{ID: orderID, RiderID: &riderID, Status: string(ordermodels.StatusOnTheWay), DeliveryCode: &code}
	m.repo.On("SearchOrdersByID", mock.Anything, orderID).Return(order, nil)

	_, err := svc.ConfirmDelivery(context.Background(), orderID, "0000", riderID)

	assert.ErrorIs(t, err, errcustom.ErrInvalidCode)
}

func TestGetOrderHistory_Success(t *testing.T) {
	svc, m := newOrderAppFixture(t)
	customerID, orderID := uuid.New(), uuid.New()
	m.repo.On("SearchOrderHistoryByCustomer", mock.Anything, customerID).Return([]ordermodels.Order{
		{ID: orderID, Number: 1, Status: string(ordermodels.StatusCompleted), TotalPrice: 22},
	}, nil)
	m.repo.On("ListStatusHistory", mock.Anything, []uuid.UUID{orderID}).Return(map[uuid.UUID][]ordermodels.StatusChange{}, nil)

	history, err := svc.GetOrderHistory(context.Background(), customerID)

	require.NoError(t, err)
	require.Len(t, history, 1)
	assert.Equal(t, orderID, history[0].ID)
	assert.Nil(t, history[0].StatusHistory)
	assert.Empty(t, history[0].Items)
}

func TestGetOrderHistory_EnrichedFromDetails(t *testing.T) {
	svc, m := newOrderAppFixture(t)
	customerID, orderID, itemID, addID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	details := fmt.Sprintf(`{"shop":{"name":"Pizzeria","logo":"l.png","phone":"0414"},"items":[{"id":%q,"name":"Pizza","amount":2,"additions":[{"id":%q,"name":"Queso","amount":3}],"flavors":[]}],"payment":{"status":"approved","bcv_rate":10}}`, itemID, addID)
	at := time.Now()
	m.repo.On("SearchOrderHistoryByCustomer", mock.Anything, customerID).Return([]ordermodels.Order{
		{ID: orderID, TotalPrice: 5, DeliveryFee: 1, Details: json.RawMessage(details)},
	}, nil)
	m.repo.On("ListStatusHistory", mock.Anything, []uuid.UUID{orderID}).Return(map[uuid.UUID][]ordermodels.StatusChange{
		orderID: {{OrderID: orderID, Status: "Pending", At: at}},
	}, nil)

	history, err := svc.GetOrderHistory(context.Background(), customerID)

	require.NoError(t, err)
	require.Len(t, history, 1)
	assert.Equal(t, "Pizzeria", history[0].ShopName)
	assert.Equal(t, 5, history[0].ProductsCount)
	require.Len(t, history[0].Items, 1)
	require.Len(t, history[0].Items[0].Additions, 1)
	require.Len(t, history[0].StatusHistory, 1)
	require.NotNil(t, history[0].TotalPriceBs)
	assert.Equal(t, 50.0, *history[0].TotalPriceBs)
}

func TestGetOrderHistory_RepoError(t *testing.T) {
	svc, m := newOrderAppFixture(t)
	customerID := uuid.New()
	m.repo.On("SearchOrderHistoryByCustomer", mock.Anything, customerID).Return(nil, errcustom.ErrUnexpectedError)

	_, err := svc.GetOrderHistory(context.Background(), customerID)

	assert.ErrorIs(t, err, errcustom.ErrUnexpectedError)
}

func TestSearchOrdersByID_DelegatesToRepo(t *testing.T) {
	svc, m := newOrderAppFixture(t)
	orderID := uuid.New()
	m.repo.On("SearchOrdersByID", mock.Anything, orderID).Return(&ordermodels.Order{ID: orderID}, nil)

	out, err := svc.SearchOrdersByID(context.Background(), orderID)

	require.NoError(t, err)
	assert.Equal(t, orderID, out.ID)
}

func TestSearchActiveOrdersByCriteria_DelegatesToRepo(t *testing.T) {
	svc, m := newOrderAppFixture(t)
	shopID := uuid.New()
	m.repo.On("SearchActiveOrdersByCriteria", mock.Anything, shopID, true).Return([]ordermodels.Order{{ShopID: shopID}}, nil)

	out, err := svc.SearchActiveOrdersByCriteria(context.Background(), shopID, true)

	require.NoError(t, err)
	require.Len(t, out, 1)
}
