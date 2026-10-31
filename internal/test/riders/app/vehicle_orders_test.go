package app_test

import (
	"context"
	"encoding/json"
	"testing"

	mockbcvports "Encargalo.app-api.go/internal/mocks/bcv/domain/ports"
	mocksriders "Encargalo.app-api.go/internal/mocks/riders/domain/ports"
	ordersModels "Encargalo.app-api.go/internal/orders/domain/models"
	"Encargalo.app-api.go/internal/riders/app"
	riderModels "Encargalo.app-api.go/internal/riders/domain/models"
	ridersPorts "Encargalo.app-api.go/internal/riders/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	shopsmodels "Encargalo.app-api.go/internal/shops/domain/models/shops"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestVehicleCanCarryShopType(t *testing.T) {
	cases := []struct {
		vehicle, shop string
		want          bool
	}{
		{riderModels.VehicleTypeBike, shopsmodels.ShopTypeRestaurant, true},
		{riderModels.VehicleTypeBike, shopsmodels.ShopTypeTurbo, false},
		{riderModels.VehicleTypeBike, shopsmodels.ShopTypeSupermarket, false},
		{riderModels.VehicleTypeMoto, shopsmodels.ShopTypeTurbo, true},
		{riderModels.VehicleTypeCar, shopsmodels.ShopTypeSupermarket, true},
		// Ni el vehículo sin cargar ni un tipo de shop desconocido excluyen.
		{"", shopsmodels.ShopTypeTurbo, true},
		{riderModels.VehicleTypeBike, "", true},
	}

	for _, c := range cases {
		require.Equal(t, c.want, riderModels.VehicleCanCarryShopType(c.vehicle, c.shop), "vehículo %q, shop %q", c.vehicle, c.shop)
	}
}

// ordersMgr arma un manager para los caminos de pedidos, con el repo expuesto para fijar expectativas.
func ordersMgr(t *testing.T) (ridersPorts.RidersApp, *mocksriders.RidersRepo, *mocksriders.VehicleLookup) {
	t.Helper()
	repo := mocksriders.NewRidersRepo(t)
	vehicles := mocksriders.NewVehicleLookup(t)
	bcv := mockbcvports.NewBcvApp(t)
	bcv.On("GetRate", mock.Anything).Return(nil, errcustom.ErrBcvRateUnavailable).Maybe()
	svc := app.New(repo, newRidersStream(t), bcv, mocksriders.NewTripsStore(t), mocksriders.NewTripEarningRecorder(t), nil, nil, nil, vehicles, newPresenceStore(t), newOfferTimeoutStore(t), nil, nil, nil, nil, nil, nil)
	return svc, repo, vehicles
}

// connectOrdersRider conecta un rider en modo órdenes sin pedidos de arranque y devuelve lo que el servidor le manda.
func connectOrdersRider(t *testing.T, svc ridersPorts.RidersApp, repo *mocksriders.RidersRepo, vehicles *mocksriders.VehicleLookup, vehicle *string) (uuid.UUID, chan []byte) {
	t.Helper()
	id := uuid.New()
	vehicles.On("GetByUserID", mock.Anything, id).Return(&riderModels.RiderProfile{UserID: id, VehicleType: vehicle}, nil).Once()
	repo.On("SearchActiveOrdersByRiderID", mock.Anything, id).Return(nil, errcustom.ErrNotFound).Once()
	repo.On("SearchAvailableOrders", mock.Anything).Return(nil, errcustom.ErrNotFound).Once()

	sent := make(chan []byte, 8)
	conn := mocksriders.NewRiderConn(t)
	conn.On("Send", mock.Anything).Run(func(args mock.Arguments) {
		sent <- args.Get(0).([]byte)
	}).Return().Maybe()

	require.NoError(t, svc.Register(context.Background(), id, conn, riderModels.RiderModeOrders))
	return id, sent
}

func orderForShop(shopID uuid.UUID) ordersModels.Order {
	return ordersModels.Order{ID: uuid.New(), ShopID: shopID, CustomerID: uuid.New(), Status: string(ordersModels.StatusReady)}
}

// --- Camino 3: AcceptOrder. Acá vive la regla; los filtros de broadcast son UX. ---

func TestAcceptOrder_BikeCannotTakeTurboOrder(t *testing.T) {
	svc, repo, vehicles := ordersMgr(t)
	riderID, _ := connectOrdersRider(t, svc, repo, vehicles, new(riderModels.VehicleTypeBike))

	order := orderForShop(uuid.New())
	repo.On("SearchOrdersByID", mock.Anything, order.ID).Return(&order, nil).Once()
	repo.On("SearchShopByID", mock.Anything, order.ShopID).Return(&ordersModels.ShopOrder{ID: order.ShopID, Type: shopsmodels.ShopTypeTurbo}, nil).Once()

	// Sin expectativa de AssignRider: si el guard no corta, mockery falla el test.
	err := svc.AcceptOrder(context.Background(), riderID, order.ID, mocksriders.NewRiderConn(t))

	require.ErrorIs(t, err, errcustom.ErrVehicleCannotCarryOrder)
}

func TestAcceptOrder_BikeCannotTakeSupermarketOrder(t *testing.T) {
	svc, repo, vehicles := ordersMgr(t)
	riderID, _ := connectOrdersRider(t, svc, repo, vehicles, new(riderModels.VehicleTypeBike))

	order := orderForShop(uuid.New())
	repo.On("SearchOrdersByID", mock.Anything, order.ID).Return(&order, nil).Once()
	repo.On("SearchShopByID", mock.Anything, order.ShopID).Return(&ordersModels.ShopOrder{ID: order.ShopID, Type: shopsmodels.ShopTypeSupermarket}, nil).Once()

	err := svc.AcceptOrder(context.Background(), riderID, order.ID, mocksriders.NewRiderConn(t))

	require.ErrorIs(t, err, errcustom.ErrVehicleCannotCarryOrder)
}

// La bici sí lleva restaurante: el guard la deja pasar y el fallo sale de AssignRider.
func TestAcceptOrder_BikeTakesRestaurantOrder(t *testing.T) {
	svc, repo, vehicles := ordersMgr(t)
	riderID, _ := connectOrdersRider(t, svc, repo, vehicles, new(riderModels.VehicleTypeBike))

	order := orderForShop(uuid.New())
	repo.On("SearchOrdersByID", mock.Anything, order.ID).Return(&order, nil).Once()
	repo.On("SearchShopByID", mock.Anything, order.ShopID).Return(&ordersModels.ShopOrder{ID: order.ShopID, Type: shopsmodels.ShopTypeRestaurant}, nil).Once()
	repo.On("AssignRider", mock.Anything, order.ID, riderID).Return(errcustom.ErrUnexpectedError).Once()

	err := svc.AcceptOrder(context.Background(), riderID, order.ID, mocksriders.NewRiderConn(t))

	require.NotErrorIs(t, err, errcustom.ErrVehicleCannotCarryOrder)
}

func TestAcceptOrder_MotoTakesTurboOrder(t *testing.T) {
	svc, repo, vehicles := ordersMgr(t)
	riderID, _ := connectOrdersRider(t, svc, repo, vehicles, new(riderModels.VehicleTypeMoto))

	order := orderForShop(uuid.New())
	repo.On("SearchOrdersByID", mock.Anything, order.ID).Return(&order, nil).Once()
	repo.On("SearchShopByID", mock.Anything, order.ShopID).Return(&ordersModels.ShopOrder{ID: order.ShopID, Type: shopsmodels.ShopTypeTurbo}, nil).Once()
	repo.On("AssignRider", mock.Anything, order.ID, riderID).Return(errcustom.ErrUnexpectedError).Once()

	err := svc.AcceptOrder(context.Background(), riderID, order.ID, mocksriders.NewRiderConn(t))

	require.NotErrorIs(t, err, errcustom.ErrVehicleCannotCarryOrder)
}

// Un shop que no se pudo leer no bloquea: mismo fail-open que el vehículo sin cargar.
func TestAcceptOrder_ShopLookupFailureDoesNotBlockTheBike(t *testing.T) {
	svc, repo, vehicles := ordersMgr(t)
	riderID, _ := connectOrdersRider(t, svc, repo, vehicles, new(riderModels.VehicleTypeBike))

	order := orderForShop(uuid.New())
	repo.On("SearchOrdersByID", mock.Anything, order.ID).Return(&order, nil).Once()
	repo.On("SearchShopByID", mock.Anything, order.ShopID).Return(nil, errcustom.ErrNotFound).Once()
	repo.On("AssignRider", mock.Anything, order.ID, riderID).Return(errcustom.ErrUnexpectedError).Once()

	err := svc.AcceptOrder(context.Background(), riderID, order.ID, mocksriders.NewRiderConn(t))

	require.NotErrorIs(t, err, errcustom.ErrVehicleCannotCarryOrder)
}

// --- Camino 1: BroadcastNewOrder ---

func TestBroadcastNewOrder_TurboSkipsBikeRiders(t *testing.T) {
	svc, repo, vehicles := ordersMgr(t)
	_, motoSent := connectOrdersRider(t, svc, repo, vehicles, new(riderModels.VehicleTypeMoto))

	// El conn de la bici no declara Send: si recibe el pedido Turbo, mockery falla el test.
	bikeID := uuid.New()
	vehicles.On("GetByUserID", mock.Anything, bikeID).Return(&riderModels.RiderProfile{UserID: bikeID, VehicleType: new(riderModels.VehicleTypeBike)}, nil).Once()
	repo.On("SearchActiveOrdersByRiderID", mock.Anything, bikeID).Return(nil, errcustom.ErrNotFound).Once()
	repo.On("SearchAvailableOrders", mock.Anything).Return(nil, errcustom.ErrNotFound).Once()
	require.NoError(t, svc.Register(context.Background(), bikeID, mocksriders.NewRiderConn(t), riderModels.RiderModeOrders))

	order := orderForShop(uuid.New())
	repo.On("SearchShopByID", mock.Anything, order.ShopID).Return(&ordersModels.ShopOrder{ID: order.ShopID, Type: shopsmodels.ShopTypeTurbo}, nil).Once()
	repo.On("SearchOrdersByID", mock.Anything, order.ID).Return(&order, nil).Maybe()

	asBroadcaster(t, svc).BroadcastNewOrder(context.Background(), order)

	require.Equal(t, order.ID.String(), orderIDOf(t, <-motoSent))
}

func TestBroadcastNewOrder_RestaurantReachesBikeRiders(t *testing.T) {
	svc, repo, vehicles := ordersMgr(t)
	_, bikeSent := connectOrdersRider(t, svc, repo, vehicles, new(riderModels.VehicleTypeBike))

	order := orderForShop(uuid.New())
	repo.On("SearchShopByID", mock.Anything, order.ShopID).Return(&ordersModels.ShopOrder{ID: order.ShopID, Type: shopsmodels.ShopTypeRestaurant}, nil).Once()
	repo.On("SearchOrdersByID", mock.Anything, order.ID).Return(&order, nil).Maybe()

	asBroadcaster(t, svc).BroadcastNewOrder(context.Background(), order)

	require.Equal(t, order.ID.String(), orderIDOf(t, <-bikeSent))
}

// --- Camino 2: el backfill de Register ---

func TestRegister_BackfillSkipsTurboForBikeRiders(t *testing.T) {
	svc, repo, vehicles := ordersMgr(t)

	turbo := orderForShop(uuid.New())
	restaurant := orderForShop(uuid.New())
	repo.On("SearchShopByID", mock.Anything, turbo.ShopID).Return(&ordersModels.ShopOrder{ID: turbo.ShopID, Type: shopsmodels.ShopTypeTurbo}, nil).Once()
	repo.On("SearchShopByID", mock.Anything, restaurant.ShopID).Return(&ordersModels.ShopOrder{ID: restaurant.ShopID, Type: shopsmodels.ShopTypeRestaurant}, nil).Once()

	bikeID := uuid.New()
	vehicles.On("GetByUserID", mock.Anything, bikeID).Return(&riderModels.RiderProfile{UserID: bikeID, VehicleType: new(riderModels.VehicleTypeBike)}, nil).Once()
	repo.On("SearchActiveOrdersByRiderID", mock.Anything, bikeID).Return(nil, errcustom.ErrNotFound).Once()
	repo.On("SearchAvailableOrders", mock.Anything).Return([]ordersModels.Order{turbo, restaurant}, nil).Once()

	sent := make(chan []byte, 8)
	conn := mocksriders.NewRiderConn(t)
	conn.On("Send", mock.Anything).Run(func(args mock.Arguments) {
		sent <- args.Get(0).([]byte)
	}).Return()

	require.NoError(t, svc.Register(context.Background(), bikeID, conn, riderModels.RiderModeOrders))

	require.Len(t, sent, 1)
	require.Equal(t, restaurant.ID.String(), orderIDOf(t, <-sent))
}

func orderIDOf(t *testing.T, raw []byte) string {
	t.Helper()
	var msg struct {
		Order struct {
			ID string `json:"id"`
		} `json:"order"`
	}
	require.NoError(t, json.Unmarshal(raw, &msg))
	return msg.Order.ID
}
