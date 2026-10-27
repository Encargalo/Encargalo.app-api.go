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

	source.On("FetchShops", mock.Anything, from, to).Return(shops, nil)
	source.On("FetchUsers", mock.Anything, from, to).Return(users, nil)
	source.On("FetchRiders", mock.Anything, from, to).Return(riders, nil)
	source.On("FetchOrders", mock.Anything, from, to).Return(orders, nil)
	source.On("FetchTrips", mock.Anything, from, to).Return(trips, nil)
	source.On("FetchTrackingEvents", mock.Anything, from, to).Return(events, nil)
	source.On("FetchFareQuotes", mock.Anything, from, to).Return(quotes, nil)
	source.On("FetchDeliveryQuotes", mock.Anything, from, to).Return(deliveryQuotes, nil)

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

	syncLog.On("Upsert", mock.Anything, models.EntityDimShop, to).Return(nil)
	syncLog.On("Upsert", mock.Anything, models.EntityDimUser, to).Return(nil)
	syncLog.On("Upsert", mock.Anything, models.EntityDimRider, to).Return(nil)
	syncLog.On("Upsert", mock.Anything, models.EntityCubeOrder, to).Return(nil)
	syncLog.On("Upsert", mock.Anything, models.EntityCubeTrip, to).Return(nil)
	syncLog.On("Upsert", mock.Anything, models.EntityFactTrackingEvent, to).Return(nil)
	syncLog.On("Upsert", mock.Anything, models.EntityCubeFareQuote, to).Return(nil)
	syncLog.On("Upsert", mock.Anything, models.EntityCubeDeliveryQuote, to).Return(nil)

	etl := newETLApp(source, sink, syncLog)
	result, err := etl.Run(context.Background(), from, to, nil)
	require.NoError(t, err)
	require.Equal(t, 8, len(result.Results))
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

	source.On("FetchOrderByID", mock.Anything, orderID).Return(order, nil)
	sink.On("UpsertOrders", mock.Anything, mock.MatchedBy(func(rows []models.CubeOrder) bool {
		return len(rows) == 1 && rows[0].OrderID == orderID && rows[0].GMV == 90
	})).Return(nil)

	etl := newETLApp(source, sink, syncLog)
	err := etl.SyncOrderByID(context.Background(), orderID)

	require.NoError(t, err)
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
