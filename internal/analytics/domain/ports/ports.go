package ports

import (
	"context"
	"time"

	"Encargalo.app-api.go/internal/analytics/domain/models"
)

type ETLApp interface {
	Run(ctx context.Context, from, to time.Time, entities []models.ETLEntity) (models.BatchResult, error)
	ResolveDateRange(ctx context.Context, from, to *time.Time) (models.DateRange, error)
}

type SyncLogRepo interface {
	GetLastSyncedTo(ctx context.Context, entity models.ETLEntity) (*time.Time, error)
	Upsert(ctx context.Context, entity models.ETLEntity, syncedTo time.Time) error
}

type OLTPSourceRepo interface {
	FetchShops(ctx context.Context, from, to time.Time) ([]models.DimShop, error)
	FetchUsers(ctx context.Context, from, to time.Time) ([]models.DimUser, error)
	FetchRiders(ctx context.Context, from, to time.Time) ([]models.DimRider, error)
	FetchOrders(ctx context.Context, from, to time.Time) ([]models.CubeOrder, error)
	FetchTrips(ctx context.Context, from, to time.Time) ([]models.CubeTrip, error)
	FetchTrackingEvents(ctx context.Context, from, to time.Time) ([]models.FactTrackingEvent, error)
	FetchFareQuotes(ctx context.Context, from, to time.Time) ([]models.CubeFareQuote, error)
}

type OLAPSinkRepo interface {
	UpsertShops(ctx context.Context, rows []models.DimShop) error
	UpsertUsers(ctx context.Context, rows []models.DimUser) error
	UpsertRiders(ctx context.Context, rows []models.DimRider) error
	UpsertOrders(ctx context.Context, rows []models.CubeOrder) error
	UpsertTrips(ctx context.Context, rows []models.CubeTrip) error
	UpsertTrackingEvents(ctx context.Context, rows []models.FactTrackingEvent) error
	UpsertFareQuotes(ctx context.Context, rows []models.CubeFareQuote) error
}
