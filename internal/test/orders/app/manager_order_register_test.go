package app_test

import (
	"context"
	"testing"

	mockbcvports "Encargalo.app-api.go/internal/mocks/bcv/domain/ports"
	mockordersports "Encargalo.app-api.go/internal/mocks/orders/domain/ports"
	"Encargalo.app-api.go/internal/orders/app"
	ordermodels "Encargalo.app-api.go/internal/orders/domain/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestManagerOrderApp_Register_SendsActiveOrdersSnapshot(t *testing.T) {
	svcMock := mockordersports.NewOrdersApp(t)
	bcvMock := mockbcvports.NewBcvApp(t)
	mgr := app.NewManagerOrderApp(svcMock, bcvMock)

	shopID := uuid.New()
	conn := mockordersports.NewManagerConn(t)

	svcMock.On("SearchActiveOrdersByCriteria", mock.Anything, shopID, true).Return([]ordermodels.Order{{ShopID: shopID, TotalPrice: 22}}, nil)
	bcvMock.On("GetRate", mock.Anything).Return(nil, assert.AnError)
	conn.On("Send", mock.Anything).Return()

	err := mgr.Register(context.Background(), shopID, conn, true)

	require.NoError(t, err)
}

func TestManagerOrderApp_Register_PropagatesUnexpectedError(t *testing.T) {
	svcMock := mockordersports.NewOrdersApp(t)
	bcvMock := mockbcvports.NewBcvApp(t)
	mgr := app.NewManagerOrderApp(svcMock, bcvMock)

	shopID := uuid.New()
	conn := mockordersports.NewManagerConn(t)

	svcMock.On("SearchActiveOrdersByCriteria", mock.Anything, shopID, true).Return(nil, assert.AnError)

	err := mgr.Register(context.Background(), shopID, conn, true)

	assert.ErrorIs(t, err, assert.AnError)
}

func TestManagerOrderApp_UnregisterRemovesConnection(t *testing.T) {
	svcMock := mockordersports.NewOrdersApp(t)
	bcvMock := mockbcvports.NewBcvApp(t)
	mgr := app.NewManagerOrderApp(svcMock, bcvMock)

	shopID := uuid.New()
	conn := mockordersports.NewManagerConn(t)
	svcMock.On("SearchActiveOrdersByCriteria", mock.Anything, shopID, true).Return(nil, nil)
	require.NoError(t, mgr.Register(context.Background(), shopID, conn, true))

	conn.On("Close").Return()
	mgr.Unregister(shopID, conn, true)

	// Tras Unregister, BroadcastShopStatus no debe alcanzar la conexión
	// removida: si Send se llamara, el mock (sin expectativa) haría panic.
	err := mgr.BroadcastShopStatus(shopID, true)
	require.NoError(t, err)
}

func TestManagerOrderApp_BroadcastShopStatus_SendsToShopConnections(t *testing.T) {
	svcMock := mockordersports.NewOrdersApp(t)
	bcvMock := mockbcvports.NewBcvApp(t)
	mgr := app.NewManagerOrderApp(svcMock, bcvMock)

	shopID := uuid.New()
	conn := mockordersports.NewManagerConn(t)
	svcMock.On("SearchActiveOrdersByCriteria", mock.Anything, shopID, true).Return(nil, nil)
	require.NoError(t, mgr.Register(context.Background(), shopID, conn, true))

	conn.On("Send", mock.Anything).Return()
	err := mgr.BroadcastShopStatus(shopID, true)

	require.NoError(t, err)
}

func TestManagerOrderApp_BroadcastOrder_SendsToShopAndCustomer(t *testing.T) {
	svcMock := mockordersports.NewOrdersApp(t)
	bcvMock := mockbcvports.NewBcvApp(t)
	mgr := app.NewManagerOrderApp(svcMock, bcvMock)

	shopID, customerID := uuid.New(), uuid.New()
	shopConn := mockordersports.NewManagerConn(t)
	customerConn := mockordersports.NewManagerConn(t)
	svcMock.On("SearchActiveOrdersByCriteria", mock.Anything, shopID, true).Return(nil, nil)
	svcMock.On("SearchActiveOrdersByCriteria", mock.Anything, customerID, false).Return(nil, nil)
	require.NoError(t, mgr.Register(context.Background(), shopID, shopConn, true))
	require.NoError(t, mgr.Register(context.Background(), customerID, customerConn, false))

	shopConn.On("Send", mock.Anything).Return()
	customerConn.On("Send", mock.Anything).Return()

	err := mgr.BroadcastOrder(ordermodels.Order{ShopID: shopID, CustomerID: customerID, TotalPrice: 22})

	require.NoError(t, err)
}

func TestManagerOrderApp_BroadcastOrderToCustomer_SendsOnlyToCustomer(t *testing.T) {
	svcMock := mockordersports.NewOrdersApp(t)
	bcvMock := mockbcvports.NewBcvApp(t)
	mgr := app.NewManagerOrderApp(svcMock, bcvMock)

	customerID := uuid.New()
	customerConn := mockordersports.NewManagerConn(t)
	svcMock.On("SearchActiveOrdersByCriteria", mock.Anything, customerID, false).Return(nil, nil)
	require.NoError(t, mgr.Register(context.Background(), customerID, customerConn, false))

	customerConn.On("Send", mock.Anything).Return()
	err := mgr.BroadcastOrderToCustomer(ordermodels.Order{CustomerID: customerID})

	require.NoError(t, err)
}

func TestManagerOrderApp_NotifyCustomer_SendsRawMessage(t *testing.T) {
	svcMock := mockordersports.NewOrdersApp(t)
	bcvMock := mockbcvports.NewBcvApp(t)
	mgr := app.NewManagerOrderApp(svcMock, bcvMock)

	customerID := uuid.New()
	customerConn := mockordersports.NewManagerConn(t)
	svcMock.On("SearchActiveOrdersByCriteria", mock.Anything, customerID, false).Return(nil, nil)
	require.NoError(t, mgr.Register(context.Background(), customerID, customerConn, false))

	payload := []byte(`{"type":"refund_request_update"}`)
	customerConn.On("Send", payload).Return()

	err := mgr.NotifyCustomer(customerID, payload)

	require.NoError(t, err)
}
