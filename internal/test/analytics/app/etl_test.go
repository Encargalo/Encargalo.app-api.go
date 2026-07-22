package app_test

import (
	"context"
	"testing"
	"time"

	"Encargalo.app-api.go/internal/analytics/app"
	"Encargalo.app-api.go/internal/analytics/domain/models"
	"Encargalo.app-api.go/internal/analytics/domain/ports"
	mocks "Encargalo.app-api.go/internal/mocks/analytics/domain/ports"
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

	shops := []models.DimShop{{ShopID: shopID, Name: "A"}}
	users := []models.DimUser{{UserID: userID, Name: "U", Roles: []string{"Customer"}}}
	riders := []models.DimRider{{RiderID: riderID, Name: "R", Status: "active"}}
	orders := []models.CubeOrder{{
		OrderID: orderID, ShopID: shopID, CustomerID: userID,
		Status: "Completed", TotalPrice: 100, DeliveryFee: 10,
	}}

	from := time.Now().Add(-time.Hour)
	to := time.Now()

	source.On("FetchShops", mock.Anything, from, to).Return(shops, nil)
	source.On("FetchUsers", mock.Anything, from, to).Return(users, nil)
	source.On("FetchRiders", mock.Anything, from, to).Return(riders, nil)
	source.On("FetchOrders", mock.Anything, from, to).Return(orders, nil)

	sink.On("UpsertShops", mock.Anything, shops).Return(nil)
	sink.On("UpsertUsers", mock.Anything, users).Return(nil)
	sink.On("UpsertRiders", mock.Anything, riders).Return(nil)
	sink.On("UpsertOrders", mock.Anything, mock.MatchedBy(func(rows []models.CubeOrder) bool {
		return len(rows) == 1 && rows[0].GMV == 90
	})).Return(nil)

	syncLog.On("Upsert", mock.Anything, models.EntityDimShop, to).Return(nil)
	syncLog.On("Upsert", mock.Anything, models.EntityDimUser, to).Return(nil)
	syncLog.On("Upsert", mock.Anything, models.EntityDimRider, to).Return(nil)
	syncLog.On("Upsert", mock.Anything, models.EntityCubeOrder, to).Return(nil)

	etl := newETLApp(source, sink, syncLog)
	result, err := etl.Run(context.Background(), from, to, nil)
	require.NoError(t, err)
	require.Equal(t, 4, len(result.Results))
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
