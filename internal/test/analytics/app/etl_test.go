package app_test

import (
	"context"
	"testing"
	"time"

	"Encargalo.app-api.go/internal/analytics/app"
	"Encargalo.app-api.go/internal/analytics/domain/models"
	"Encargalo.app-api.go/internal/analytics/domain/ports"
	mocks "Encargalo.app-api.go/internal/mocks/analytics/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newETLApp(source *mocks.OLTPSourceRepo, sink *mocks.OLAPSinkRepo, sync *mocks.SyncLogRepo) ports.ETLApp {
	return app.NewETLApp(source, sink, sync, app.Lookbacks{
		Default: 48 * time.Hour,
		Initial: 5 * 365 * 24 * time.Hour,
	})
}

func TestResolveDateRange_InitialLookback(t *testing.T) {
	source := new(mocks.OLTPSourceRepo)
	sink := new(mocks.OLAPSinkRepo)
	syncLog := new(mocks.SyncLogRepo)
	syncLog.On("GetLastSyncedTo", mock.Anything, models.EntityCubeOrder).Return((*time.Time)(nil), nil)

	etl := newETLApp(source, sink, syncLog)
	r, err := etl.ResolveDateRange(context.Background(), nil, nil)
	require.NoError(t, err)
	require.True(t, r.From.Before(r.To))
	require.Greater(t, r.To.Sub(r.From), 48*time.Hour)
	syncLog.AssertExpectations(t)
}

func TestResolveDateRange_WithWatermark(t *testing.T) {
	now := time.Now().UTC()
	last := now.Add(-2 * time.Hour)
	source := new(mocks.OLTPSourceRepo)
	sink := new(mocks.OLAPSinkRepo)
	syncLog := new(mocks.SyncLogRepo)
	syncLog.On("GetLastSyncedTo", mock.Anything, models.EntityCubeOrder).Return(&last, nil)

	etl := newETLApp(source, sink, syncLog)
	r, err := etl.ResolveDateRange(context.Background(), nil, nil)
	require.NoError(t, err)
	require.WithinDuration(t, last.Add(-48*time.Hour), r.From, time.Second)
	syncLog.AssertExpectations(t)
}

func TestRun_SyncsAllEntities(t *testing.T) {
	source := new(mocks.OLTPSourceRepo)
	sink := new(mocks.OLAPSinkRepo)
	syncLog := new(mocks.SyncLogRepo)

	shopID := uuid.New()
	userID := uuid.New()
	riderID := uuid.New()
	orderID := uuid.New()
	tripID := uuid.New()

	shops := []models.DimShop{{ShopID: shopID, Name: "A"}}
	users := []models.DimUser{{UserID: userID, Name: "U", Roles: []string{"Customer"}}}
	riders := []models.DimRider{{RiderID: riderID, Name: "R", Status: "active"}}
	orders := []models.CubeOrder{{
		OrderID: orderID, ShopID: shopID, CustomerID: userID,
		Status: "Completed", TotalPrice: 100, DeliveryFee: 10,
	}}
	trips := []models.CubeTrip{{
		TripID: tripID, CustomerID: userID, RiderID: &riderID, Status: "completed",
	}}

	from := time.Now().Add(-time.Hour)
	to := time.Now()

	events := []models.FactTrackingEvent{{
		ID: uuid.New(), Type: "pixel", Params: map[string]any{"name": "app_open"},
	}}

	quotes := []models.CubeFareQuote{{
		QuoteID: uuid.New(), CustomerID: userID, DistanceMeters: 4270,
		DurationMin: 12, FareUSD: 1, FareBand: "day", QuoteHour: 12,
	}}

	deliveryQuotes := []models.CubeDeliveryQuote{{
		QuoteID: uuid.New(), CustomerID: userID, ShopID: shopID, DistanceMeters: 2000,
		FeeUSD: 1, FareBand: "day", QuoteHour: 12,
	}}

	statusChanges := []models.FactOrderStatusChange{{
		ID: 1, OrderID: orderID, Status: "Completed", At: to,
	}}

	source.On("FetchShops", mock.Anything, from, to).Return(shops, nil)
	source.On("FetchUsers", mock.Anything, from, to).Return(users, nil)
	source.On("FetchRiders", mock.Anything, from, to).Return(riders, nil)
	source.On("FetchOrders", mock.Anything, from, to).Return(orders, nil)
	source.On("FetchTrips", mock.Anything, from, to).Return(trips, nil)
	source.On("FetchTrackingEvents", mock.Anything, from, to).Return(events, nil)
	source.On("FetchFareQuotes", mock.Anything, from, to).Return(quotes, nil)
	source.On("FetchDeliveryQuotes", mock.Anything, from, to).Return(deliveryQuotes, nil)
	source.On("FetchOrderStatusChanges", mock.Anything, from, to).Return(statusChanges, nil)

	sink.On("UpsertShops", mock.Anything, shops).Return(nil)
	sink.On("UpsertUsers", mock.Anything, users).Return(nil)
	sink.On("UpsertRiders", mock.Anything, riders).Return(nil)
	sink.On("UpsertOrders", mock.Anything, mock.MatchedBy(func(rows []models.CubeOrder) bool {
		return len(rows) == 1 && rows[0].GMV == 90
	})).Return(nil)
	sink.On("UpsertTrips", mock.Anything, trips).Return(nil)
	sink.On("UpsertTrackingEvents", mock.Anything, events).Return(nil)
	sink.On("UpsertFareQuotes", mock.Anything, quotes).Return(nil)
	sink.On("UpsertDeliveryQuotes", mock.Anything, deliveryQuotes).Return(nil)
	sink.On("UpsertOrderStatusChanges", mock.Anything, statusChanges).Return(nil)

	syncLog.On("Upsert", mock.Anything, models.EntityDimShop, to).Return(nil)
	syncLog.On("Upsert", mock.Anything, models.EntityDimUser, to).Return(nil)
	syncLog.On("Upsert", mock.Anything, models.EntityDimRider, to).Return(nil)
	syncLog.On("Upsert", mock.Anything, models.EntityCubeOrder, to).Return(nil)
	syncLog.On("Upsert", mock.Anything, models.EntityCubeTrip, to).Return(nil)
	syncLog.On("Upsert", mock.Anything, models.EntityFactTrackingEvent, to).Return(nil)
	syncLog.On("Upsert", mock.Anything, models.EntityCubeFareQuote, to).Return(nil)
	syncLog.On("Upsert", mock.Anything, models.EntityCubeDeliveryQuote, to).Return(nil)
	syncLog.On("Upsert", mock.Anything, models.EntityFactOrderStatusChange, to).Return(nil)

	etl := newETLApp(source, sink, syncLog)
	result, err := etl.Run(context.Background(), from, to, nil)
	require.NoError(t, err)
	require.Equal(t, 9, len(result.Results))
	for _, item := range result.Results {
		require.True(t, item.Success, item.Entity)
	}
	source.AssertExpectations(t)
	sink.AssertExpectations(t)
	syncLog.AssertExpectations(t)
}

func TestGMV(t *testing.T) {
	require.Equal(t, 90.0, models.GMV(100, 10))
	require.Equal(t, 0.0, models.GMV(5, 10))
}

func TestSyncOrderByID_Success(t *testing.T) {
	source := new(mocks.OLTPSourceRepo)
	sink := new(mocks.OLAPSinkRepo)
	syncLog := new(mocks.SyncLogRepo)

	orderID := uuid.New()
	order := &models.CubeOrder{OrderID: orderID, Status: "Completed", TotalPrice: 100, DeliveryFee: 10}
	history := []models.FactOrderStatusChange{
		{ID: 1, OrderID: orderID, Status: "Pending"},
		{ID: 2, OrderID: orderID, Status: "Completed"},
	}

	source.On("FetchOrderByID", mock.Anything, orderID).Return(order, nil)
	sink.On("UpsertOrders", mock.Anything, mock.MatchedBy(func(rows []models.CubeOrder) bool {
		return len(rows) == 1 && rows[0].OrderID == orderID && rows[0].GMV == 90
	})).Return(nil)
	source.On("FetchOrderStatusChangesByOrderID", mock.Anything, orderID).Return(history, nil)
	sink.On("UpsertOrderStatusChanges", mock.Anything, history).Return(nil)

	etl := newETLApp(source, sink, syncLog)
	err := etl.SyncOrderByID(context.Background(), orderID)

	require.NoError(t, err)
	source.AssertExpectations(t)
	sink.AssertExpectations(t)
}

// insertInChunks ya no-opea en un slice vacío, así que SyncOrderByID no
// necesita un guard propio: delega el caso vacío al sink sin problema.
func TestSyncOrderByID_NoHistoryIsNoop(t *testing.T) {
	source := new(mocks.OLTPSourceRepo)
	sink := new(mocks.OLAPSinkRepo)
	syncLog := new(mocks.SyncLogRepo)

	orderID := uuid.New()
	order := &models.CubeOrder{OrderID: orderID, Status: "Pending"}

	source.On("FetchOrderByID", mock.Anything, orderID).Return(order, nil)
	sink.On("UpsertOrders", mock.Anything, mock.Anything).Return(nil)
	source.On("FetchOrderStatusChangesByOrderID", mock.Anything, orderID).Return([]models.FactOrderStatusChange{}, nil)
	sink.On("UpsertOrderStatusChanges", mock.Anything, []models.FactOrderStatusChange{}).Return(nil)

	etl := newETLApp(source, sink, syncLog)
	err := etl.SyncOrderByID(context.Background(), orderID)

	require.NoError(t, err)
	source.AssertExpectations(t)
	sink.AssertExpectations(t)
}

func TestSyncOrderByID_HistoryFetchError(t *testing.T) {
	source := new(mocks.OLTPSourceRepo)
	sink := new(mocks.OLAPSinkRepo)
	syncLog := new(mocks.SyncLogRepo)

	orderID := uuid.New()
	order := &models.CubeOrder{OrderID: orderID, Status: "Completed", TotalPrice: 100, DeliveryFee: 10}

	source.On("FetchOrderByID", mock.Anything, orderID).Return(order, nil)
	sink.On("UpsertOrders", mock.Anything, mock.Anything).Return(nil)
	source.On("FetchOrderStatusChangesByOrderID", mock.Anything, orderID).
		Return(nil, errcustom.ErrUnexpectedError)

	etl := newETLApp(source, sink, syncLog)
	err := etl.SyncOrderByID(context.Background(), orderID)

	require.ErrorIs(t, err, errcustom.ErrUnexpectedError)
	source.AssertExpectations(t)
	sink.AssertExpectations(t)
}

func TestSyncOrderByID_FetchError(t *testing.T) {
	source := new(mocks.OLTPSourceRepo)
	sink := new(mocks.OLAPSinkRepo)
	syncLog := new(mocks.SyncLogRepo)

	orderID := uuid.New()
	source.On("FetchOrderByID", mock.Anything, orderID).Return((*models.CubeOrder)(nil), errcustom.ErrNotFound)

	etl := newETLApp(source, sink, syncLog)
	err := etl.SyncOrderByID(context.Background(), orderID)

	require.ErrorIs(t, err, errcustom.ErrNotFound)
	sink.AssertNotCalled(t, "UpsertOrders", mock.Anything, mock.Anything)
}

func TestSyncFareQuoteByID_Success(t *testing.T) {
	source := new(mocks.OLTPSourceRepo)
	sink := new(mocks.OLAPSinkRepo)
	syncLog := new(mocks.SyncLogRepo)

	quoteID := uuid.New()
	quote := &models.CubeFareQuote{QuoteID: quoteID, FareUSD: 1, FareBand: "day"}

	source.On("FetchFareQuoteByID", mock.Anything, quoteID).Return(quote, nil)
	sink.On("UpsertFareQuotes", mock.Anything, []models.CubeFareQuote{*quote}).Return(nil)

	etl := newETLApp(source, sink, syncLog)
	err := etl.SyncFareQuoteByID(context.Background(), quoteID)

	require.NoError(t, err)
	source.AssertExpectations(t)
	sink.AssertExpectations(t)
}

func TestSyncFareQuoteByID_FetchError(t *testing.T) {
	source := new(mocks.OLTPSourceRepo)
	sink := new(mocks.OLAPSinkRepo)
	syncLog := new(mocks.SyncLogRepo)

	quoteID := uuid.New()
	source.On("FetchFareQuoteByID", mock.Anything, quoteID).Return((*models.CubeFareQuote)(nil), errcustom.ErrNotFound)

	etl := newETLApp(source, sink, syncLog)
	err := etl.SyncFareQuoteByID(context.Background(), quoteID)

	require.ErrorIs(t, err, errcustom.ErrNotFound)
	sink.AssertNotCalled(t, "UpsertFareQuotes", mock.Anything, mock.Anything)
}

func TestSyncDeliveryQuoteByID_Success(t *testing.T) {
	source := new(mocks.OLTPSourceRepo)
	sink := new(mocks.OLAPSinkRepo)
	syncLog := new(mocks.SyncLogRepo)

	quoteID := uuid.New()
	quote := &models.CubeDeliveryQuote{QuoteID: quoteID, FeeUSD: 1, FareBand: "day"}

	source.On("FetchDeliveryQuoteByID", mock.Anything, quoteID).Return(quote, nil)
	sink.On("UpsertDeliveryQuotes", mock.Anything, []models.CubeDeliveryQuote{*quote}).Return(nil)

	etl := newETLApp(source, sink, syncLog)
	err := etl.SyncDeliveryQuoteByID(context.Background(), quoteID)

	require.NoError(t, err)
	source.AssertExpectations(t)
	sink.AssertExpectations(t)
}

func TestSyncDeliveryQuoteByID_FetchError(t *testing.T) {
	source := new(mocks.OLTPSourceRepo)
	sink := new(mocks.OLAPSinkRepo)
	syncLog := new(mocks.SyncLogRepo)

	quoteID := uuid.New()
	source.On("FetchDeliveryQuoteByID", mock.Anything, quoteID).Return((*models.CubeDeliveryQuote)(nil), errcustom.ErrNotFound)

	etl := newETLApp(source, sink, syncLog)
	err := etl.SyncDeliveryQuoteByID(context.Background(), quoteID)

	require.ErrorIs(t, err, errcustom.ErrNotFound)
	sink.AssertNotCalled(t, "UpsertDeliveryQuotes", mock.Anything, mock.Anything)
}

func TestSyncTripByID_Success(t *testing.T) {
	source := new(mocks.OLTPSourceRepo)
	sink := new(mocks.OLAPSinkRepo)
	syncLog := new(mocks.SyncLogRepo)

	tripID := uuid.New()
	trip := &models.CubeTrip{TripID: tripID, Status: "completed"}

	source.On("FetchTripByID", mock.Anything, tripID).Return(trip, nil)
	sink.On("UpsertTrips", mock.Anything, []models.CubeTrip{*trip}).Return(nil)

	etl := newETLApp(source, sink, syncLog)
	err := etl.SyncTripByID(context.Background(), tripID)

	require.NoError(t, err)
	source.AssertExpectations(t)
	sink.AssertExpectations(t)
}

func TestSyncTripByID_FetchError(t *testing.T) {
	source := new(mocks.OLTPSourceRepo)
	sink := new(mocks.OLAPSinkRepo)
	syncLog := new(mocks.SyncLogRepo)

	tripID := uuid.New()
	source.On("FetchTripByID", mock.Anything, tripID).Return((*models.CubeTrip)(nil), errcustom.ErrNotFound)

	etl := newETLApp(source, sink, syncLog)
	err := etl.SyncTripByID(context.Background(), tripID)

	require.ErrorIs(t, err, errcustom.ErrNotFound)
	sink.AssertNotCalled(t, "UpsertTrips", mock.Anything, mock.Anything)
}

func TestSyncShopByID_Success(t *testing.T) {
	source := new(mocks.OLTPSourceRepo)
	sink := new(mocks.OLAPSinkRepo)
	syncLog := new(mocks.SyncLogRepo)

	shopID := uuid.New()
	shop := &models.DimShop{ShopID: shopID, Name: "Pizzería Don Luigi"}

	source.On("FetchShopByID", mock.Anything, shopID).Return(shop, nil)
	sink.On("UpsertShops", mock.Anything, []models.DimShop{*shop}).Return(nil)

	etl := newETLApp(source, sink, syncLog)
	err := etl.SyncShopByID(context.Background(), shopID)

	require.NoError(t, err)
	source.AssertExpectations(t)
	sink.AssertExpectations(t)
}

func TestSyncShopByID_FetchError(t *testing.T) {
	source := new(mocks.OLTPSourceRepo)
	sink := new(mocks.OLAPSinkRepo)
	syncLog := new(mocks.SyncLogRepo)

	shopID := uuid.New()
	source.On("FetchShopByID", mock.Anything, shopID).Return((*models.DimShop)(nil), errcustom.ErrNotFound)

	etl := newETLApp(source, sink, syncLog)
	err := etl.SyncShopByID(context.Background(), shopID)

	require.ErrorIs(t, err, errcustom.ErrNotFound)
	sink.AssertNotCalled(t, "UpsertShops", mock.Anything, mock.Anything)
}

func TestSyncRiderByID_Success(t *testing.T) {
	source := new(mocks.OLTPSourceRepo)
	sink := new(mocks.OLAPSinkRepo)
	syncLog := new(mocks.SyncLogRepo)

	riderID := uuid.New()
	rider := &models.DimRider{RiderID: riderID, Status: "active"}

	source.On("FetchRiderByID", mock.Anything, riderID).Return(rider, nil)
	sink.On("UpsertRiders", mock.Anything, []models.DimRider{*rider}).Return(nil)

	etl := newETLApp(source, sink, syncLog)
	err := etl.SyncRiderByID(context.Background(), riderID)

	require.NoError(t, err)
	source.AssertExpectations(t)
	sink.AssertExpectations(t)
}

func TestSyncRiderByID_FetchError(t *testing.T) {
	source := new(mocks.OLTPSourceRepo)
	sink := new(mocks.OLAPSinkRepo)
	syncLog := new(mocks.SyncLogRepo)

	riderID := uuid.New()
	source.On("FetchRiderByID", mock.Anything, riderID).Return((*models.DimRider)(nil), errcustom.ErrNotFound)

	etl := newETLApp(source, sink, syncLog)
	err := etl.SyncRiderByID(context.Background(), riderID)

	require.ErrorIs(t, err, errcustom.ErrNotFound)
	sink.AssertNotCalled(t, "UpsertRiders", mock.Anything, mock.Anything)
}

func TestSyncUserByID_Success(t *testing.T) {
	source := new(mocks.OLTPSourceRepo)
	sink := new(mocks.OLAPSinkRepo)
	syncLog := new(mocks.SyncLogRepo)

	userID := uuid.New()
	user := &models.DimUser{UserID: userID, Name: "Ada", Roles: []string{"Customer"}}

	source.On("FetchUserByID", mock.Anything, userID).Return(user, nil)
	sink.On("UpsertUsers", mock.Anything, []models.DimUser{*user}).Return(nil)

	etl := newETLApp(source, sink, syncLog)
	err := etl.SyncUserByID(context.Background(), userID)

	require.NoError(t, err)
	source.AssertExpectations(t)
	sink.AssertExpectations(t)
}

func TestSyncUserByID_FetchError(t *testing.T) {
	source := new(mocks.OLTPSourceRepo)
	sink := new(mocks.OLAPSinkRepo)
	syncLog := new(mocks.SyncLogRepo)

	userID := uuid.New()
	source.On("FetchUserByID", mock.Anything, userID).Return((*models.DimUser)(nil), errcustom.ErrNotFound)

	etl := newETLApp(source, sink, syncLog)
	err := etl.SyncUserByID(context.Background(), userID)

	require.ErrorIs(t, err, errcustom.ErrNotFound)
	sink.AssertNotCalled(t, "UpsertUsers", mock.Anything, mock.Anything)
}
