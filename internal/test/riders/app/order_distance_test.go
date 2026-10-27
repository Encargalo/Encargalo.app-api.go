package app_test

import (
	"context"
	"encoding/json"
	"testing"

	mockbcvports "Encargalo.app-api.go/internal/mocks/bcv/domain/ports"
	mocksriders "Encargalo.app-api.go/internal/mocks/riders/domain/ports"
	mockstrips "Encargalo.app-api.go/internal/mocks/trips/domain/ports"
	ordersModels "Encargalo.app-api.go/internal/orders/domain/models"
	"Encargalo.app-api.go/internal/pkg/geo"
	"Encargalo.app-api.go/internal/riders/app"
	riderModels "Encargalo.app-api.go/internal/riders/domain/models"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// registerCapturingConn registra un rider en modo órdenes y devuelve un canal
// con lo que el servidor le envía por su conexión.
func registerCapturingConn(t *testing.T) (*mocksriders.RidersRepo, chan []byte, func(context.Context, ordersModels.Order), func(context.Context, ordersModels.Order)) {
	t.Helper()
	repo := mocksriders.NewRidersRepo(t)
	svc := app.New(repo, newRidersStream(t), mockbcvports.NewBcvApp(t), mockstrips.NewTripsRepo(t), mocksriders.NewTripEarningRecorder(t), nil, nil, nil, nil, newPresenceStore(t))
	broadcaster := asBroadcaster(t, svc)

	sent := make(chan []byte, 4)
	conn := mocksriders.NewRiderConn(t)
	conn.On("Send", mock.Anything).Run(func(args mock.Arguments) {
		sent <- args.Get(0).([]byte)
	}).Return()

	riderID := uuid.New()
	repo.On("SearchActiveOrdersByRiderID", mock.Anything, riderID).Return(nil, errcustom.ErrNotFound)
	repo.On("SearchAvailableOrders", mock.Anything).Return(nil, errcustom.ErrNotFound)
	require.NoError(t, svc.Register(context.Background(), riderID, conn, riderModels.RiderModeOrders))

	return repo, sent, broadcaster.BroadcastNewOrder, broadcaster.BroadcastOrderUpdate
}

func TestBroadcastNewOrder_IncludesShopToCustomerKm(t *testing.T) {
	repo, sent, broadcastNew, _ := registerCapturingConn(t)

	order := ordersModels.Order{ID: uuid.New(), ShopID: uuid.New(), Status: string(ordersModels.StatusReady), Latitude: 10.50, Longitude: -66.90}
	shop := &ordersModels.ShopOrder{Latitude: 10.48, Longitude: -66.88}
	repo.On("SearchShopByID", mock.Anything, order.ShopID).Return(shop, nil)

	broadcastNew(context.Background(), order)

	var msg struct {
		ShopToCustomerKm *float64 `json:"shop_to_customer_km"`
	}
	require.NoError(t, json.Unmarshal(<-sent, &msg))
	require.NotNil(t, msg.ShopToCustomerKm)
	assert.InDelta(t, geo.HaversineKm(shop.Latitude, shop.Longitude, order.Latitude, order.Longitude), *msg.ShopToCustomerKm, 1e-9)
}

func TestBroadcastNewOrder_OmitsShopToCustomerKmWhenShopLookupFails(t *testing.T) {
	repo, sent, broadcastNew, _ := registerCapturingConn(t)

	order := ordersModels.Order{ID: uuid.New(), ShopID: uuid.New(), Status: string(ordersModels.StatusReady)}
	repo.On("SearchShopByID", mock.Anything, order.ShopID).Return(nil, errcustom.ErrNotFound)

	broadcastNew(context.Background(), order)

	var raw map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(<-sent, &raw))
	assert.NotContains(t, raw, "shop_to_customer_km")
}
