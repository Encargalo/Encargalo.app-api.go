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
	FetchOrderByID(ctx context.Context, orderID uuid.UUID) (*models.CubeOrder, error)
	FetchTrips(ctx context.Context, from, to time.Time) ([]models.CubeTrip, error)
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
