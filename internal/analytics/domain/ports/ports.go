package ports

import (
	"context"
	"time"

	"Encargalo.app-api.go/internal/analytics/domain/models"
	"github.com/google/uuid"
)

type ETLApp interface {
	Run(ctx context.Context, from, to time.Time, entities []models.ETLEntity) (models.BatchResult, error)
	ResolveDateRange(ctx context.Context, from, to *time.Time) (models.DateRange, error)
	// SyncOrderByID re-sincroniza una sola orden contra OLAP. Lo usa el
	// consumidor de eventos (cmd/etl) para reaccionar a order_status_changed
	// sin esperar al batch.
	SyncOrderByID(ctx context.Context, orderID uuid.UUID) error
	// SyncFareQuoteByID re-sincroniza una sola cotización de viaje contra OLAP,
	// en reacción a fare_quote_created.
	SyncFareQuoteByID(ctx context.Context, quoteID uuid.UUID) error
	// SyncDeliveryQuoteByID re-sincroniza una sola cotización de delivery
	// contra OLAP, en reacción a delivery_quote_created.
	SyncDeliveryQuoteByID(ctx context.Context, quoteID uuid.UUID) error
	// SyncTripByID re-sincroniza un solo viaje contra OLAP, en reacción a
	// trip_status_changed (publicado tanto desde trips/app como desde
	// riders/app, ver CLAUDE.md "Trips & Riders").
	SyncTripByID(ctx context.Context, tripID uuid.UUID) error
	// SyncShopByID re-sincroniza un solo local contra OLAP, en reacción a
	// shop_updated.
	SyncShopByID(ctx context.Context, shopID uuid.UUID) error
}

type SyncLogRepo interface {
	GetLastSyncedTo(ctx context.Context, entity models.ETLEntity) (*time.Time, error)
	Upsert(ctx context.Context, entity models.ETLEntity, syncedTo time.Time) error
}

type OLTPSourceRepo interface {
	FetchShops(ctx context.Context, from, to time.Time) ([]models.DimShop, error)
	FetchShopByID(ctx context.Context, shopID uuid.UUID) (*models.DimShop, error)
	FetchUsers(ctx context.Context, from, to time.Time) ([]models.DimUser, error)
	FetchRiders(ctx context.Context, from, to time.Time) ([]models.DimRider, error)
	FetchOrders(ctx context.Context, from, to time.Time) ([]models.CubeOrder, error)
	FetchOrderByID(ctx context.Context, orderID uuid.UUID) (*models.CubeOrder, error)
	FetchTrips(ctx context.Context, from, to time.Time) ([]models.CubeTrip, error)
	FetchTripByID(ctx context.Context, tripID uuid.UUID) (*models.CubeTrip, error)
	FetchTrackingEvents(ctx context.Context, from, to time.Time) ([]models.FactTrackingEvent, error)
	FetchFareQuotes(ctx context.Context, from, to time.Time) ([]models.CubeFareQuote, error)
	FetchFareQuoteByID(ctx context.Context, quoteID uuid.UUID) (*models.CubeFareQuote, error)
	FetchDeliveryQuotes(ctx context.Context, from, to time.Time) ([]models.CubeDeliveryQuote, error)
	FetchDeliveryQuoteByID(ctx context.Context, quoteID uuid.UUID) (*models.CubeDeliveryQuote, error)
}

type OLAPSinkRepo interface {
	UpsertShops(ctx context.Context, rows []models.DimShop) error
	UpsertUsers(ctx context.Context, rows []models.DimUser) error
	UpsertRiders(ctx context.Context, rows []models.DimRider) error
	UpsertOrders(ctx context.Context, rows []models.CubeOrder) error
	UpsertTrips(ctx context.Context, rows []models.CubeTrip) error
	UpsertTrackingEvents(ctx context.Context, rows []models.FactTrackingEvent) error
	UpsertFareQuotes(ctx context.Context, rows []models.CubeFareQuote) error
	UpsertDeliveryQuotes(ctx context.Context, rows []models.CubeDeliveryQuote) error
}
