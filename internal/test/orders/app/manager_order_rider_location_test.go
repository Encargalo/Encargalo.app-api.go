package app_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	mockbcvports "Encargalo.app-api.go/internal/mocks/bcv/domain/ports"
	mockordersports "Encargalo.app-api.go/internal/mocks/orders/domain/ports"
	"Encargalo.app-api.go/internal/orders/app"
	ordermodels "Encargalo.app-api.go/internal/orders/domain/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func sentRiderLocations(conn *mockordersports.ManagerConn) *[][]byte {
	var sent [][]byte
	conn.On("Send", mock.Anything).Run(func(args mock.Arguments) {
		sent = append(sent, args.Get(0).([]byte))
	}).Return()
	return &sent
}

func TestManagerOrderApp_NotifyRiderOrderLocation_SendsExactPayloadToCustomer(t *testing.T) {
	svcMock := mockordersports.NewOrdersApp(t)
	mgr := app.NewManagerOrderApp(svcMock, mockbcvports.NewBcvApp(t))
	customerID, orderID := uuid.New(), uuid.New()
	conn := mockordersports.NewManagerConn(t)
	svcMock.On("SearchActiveOrdersByCriteria", mock.Anything, customerID, false).Return(nil, nil)
	require.NoError(t, mgr.Register(context.Background(), customerID, conn, false))
	sent := sentRiderLocations(conn)

	mgr.NotifyRiderOrderLocation(context.Background(), customerID, orderID, 10.48, -66.9, time.Date(2026, 10, 10, 14, 3, 22, 0, time.UTC))

	require.Len(t, *sent, 1)
	assert.JSONEq(t, `{"type":"rider_location","order_id":"`+orderID.String()+`","latitude":10.48,"longitude":-66.9,"updated_at":"2026-10-10T14:03:22Z"}`, string((*sent)[0]))
}

func TestManagerOrderApp_NotifyRiderOrderLocation_ConvertsUpdatedAtToUTC(t *testing.T) {
	svcMock := mockordersports.NewOrdersApp(t)
	mgr := app.NewManagerOrderApp(svcMock, mockbcvports.NewBcvApp(t))
	customerID := uuid.New()
	conn := mockordersports.NewManagerConn(t)
	svcMock.On("SearchActiveOrdersByCriteria", mock.Anything, customerID, false).Return(nil, nil)
	require.NoError(t, mgr.Register(context.Background(), customerID, conn, false))
	sent := sentRiderLocations(conn)

	local := time.FixedZone("VET", -4*3600)
	mgr.NotifyRiderOrderLocation(context.Background(), customerID, uuid.New(), 1, 2, time.Date(2026, 10, 10, 10, 3, 22, 0, local))

	var msg map[string]any
	require.NoError(t, json.Unmarshal((*sent)[0], &msg))
	assert.Equal(t, "2026-10-10T14:03:22Z", msg["updated_at"])
}

func TestManagerOrderApp_Register_ReplaysLastRiderLocationForActiveOrder(t *testing.T) {
	svcMock := mockordersports.NewOrdersApp(t)
	bcvMock := mockbcvports.NewBcvApp(t)
	mgr := app.NewManagerOrderApp(svcMock, bcvMock)
	customerID := uuid.New()
	order := ordermodels.Order{ID: uuid.New(), CustomerID: customerID, Status: string(ordermodels.StatusOnTheWay)}
	mgr.NotifyRiderOrderLocation(context.Background(), customerID, order.ID, 10.1, -66.1, time.Now())
	mgr.NotifyRiderOrderLocation(context.Background(), customerID, order.ID, 10.2, -66.2, time.Now())
	svcMock.On("SearchActiveOrdersByCriteria", mock.Anything, customerID, false).Return([]ordermodels.Order{order}, nil)
	bcvMock.On("GetRate", mock.Anything).Return(nil, assert.AnError)
	conn := mockordersports.NewManagerConn(t)
	sent := sentRiderLocations(conn)

	require.NoError(t, mgr.Register(context.Background(), customerID, conn, false))

	require.Len(t, *sent, 2)
	var first, second map[string]any
	require.NoError(t, json.Unmarshal((*sent)[0], &first))
	require.NoError(t, json.Unmarshal((*sent)[1], &second))
	assert.Equal(t, "order_update", first["type"])
	assert.Equal(t, "rider_location", second["type"])
	assert.Equal(t, 10.2, second["latitude"])
}

func TestManagerOrderApp_Register_ShopDoesNotReceiveRiderLocation(t *testing.T) {
	svcMock := mockordersports.NewOrdersApp(t)
	bcvMock := mockbcvports.NewBcvApp(t)
	mgr := app.NewManagerOrderApp(svcMock, bcvMock)
	shopID, customerID := uuid.New(), uuid.New()
	order := ordermodels.Order{ID: uuid.New(), ShopID: shopID, CustomerID: customerID, Status: string(ordermodels.StatusOnTheWay)}
	mgr.NotifyRiderOrderLocation(context.Background(), customerID, order.ID, 10.1, -66.1, time.Now())
	svcMock.On("SearchActiveOrdersByCriteria", mock.Anything, shopID, true).Return([]ordermodels.Order{order}, nil)
	bcvMock.On("GetRate", mock.Anything).Return(nil, assert.AnError)
	conn := mockordersports.NewManagerConn(t)
	sent := sentRiderLocations(conn)

	require.NoError(t, mgr.Register(context.Background(), shopID, conn, true))

	require.Len(t, *sent, 1)
}

func TestManagerOrderApp_FinishedOrderDropsStoredRiderLocation(t *testing.T) {
	for _, status := range []ordermodels.OrderStatus{ordermodels.StatusCompleted, ordermodels.StatusCancelled, ordermodels.StatusRejected} {
		t.Run(string(status), func(t *testing.T) {
			svcMock := mockordersports.NewOrdersApp(t)
			bcvMock := mockbcvports.NewBcvApp(t)
			mgr := app.NewManagerOrderApp(svcMock, bcvMock)
			customerID := uuid.New()
			order := ordermodels.Order{ID: uuid.New(), CustomerID: customerID, Status: string(status)}
			mgr.NotifyRiderOrderLocation(context.Background(), customerID, order.ID, 10.1, -66.1, time.Now())
			require.NoError(t, mgr.BroadcastOrder(order))
			svcMock.On("SearchActiveOrdersByCriteria", mock.Anything, customerID, false).Return([]ordermodels.Order{order}, nil)
			bcvMock.On("GetRate", mock.Anything).Return(nil, assert.AnError)
			conn := mockordersports.NewManagerConn(t)
			sent := sentRiderLocations(conn)

			require.NoError(t, mgr.Register(context.Background(), customerID, conn, false))

			require.Len(t, *sent, 1)
		})
	}
}
