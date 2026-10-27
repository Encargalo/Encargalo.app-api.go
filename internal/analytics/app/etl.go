package app

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"Encargalo.app-api.go/internal/analytics/domain/models"
	"Encargalo.app-api.go/internal/analytics/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/google/uuid"
)

type Lookbacks struct {
	Default time.Duration
	Initial time.Duration
}

func (l Lookbacks) withDefaults() Lookbacks {
	out := l
	if out.Default <= 0 {
		out.Default = 48 * time.Hour
	}
	if out.Initial <= 0 {
		out.Initial = 365 * 24 * time.Hour * 5
	}
	return out
}

type etlApp struct {
	source    ports.OLTPSourceRepo
	sink      ports.OLAPSinkRepo
	syncLog   ports.SyncLogRepo
	lookbacks Lookbacks
}

func NewETLApp(source ports.OLTPSourceRepo, sink ports.OLAPSinkRepo, syncLog ports.SyncLogRepo, lookbacks Lookbacks) ports.ETLApp {
	return &etlApp{
		source:    source,
		sink:      sink,
		syncLog:   syncLog,
		lookbacks: lookbacks.withDefaults(),
	}
}

func (a *etlApp) ResolveDateRange(ctx context.Context, from, to *time.Time) (models.DateRange, error) {
	now := time.Now().UTC()
	end := now
	if to != nil {
		end = to.UTC()
	}

	start := end.Add(-a.lookbacks.Initial)
	if from != nil {
		start = from.UTC()
	} else {
		last, err := a.syncLog.GetLastSyncedTo(ctx, models.EntityCubeOrder)
		if err != nil {
			return models.DateRange{}, err
		}
		if last != nil {
			start = last.Add(-a.lookbacks.Default)
		}
	}

	if !start.Before(end) {
		return models.DateRange{}, errcustom.ErrInvalidDateRange
	}

	return models.DateRange{From: start, To: end}, nil
}

func (a *etlApp) Run(ctx context.Context, from, to time.Time, entities []models.ETLEntity) (models.BatchResult, error) {
	if len(entities) == 0 {
		entities = models.DefaultETLEntities
	}

	result := models.BatchResult{
		From:    from,
		To:      to,
		Results: make([]models.ETLEntityResult, 0, len(entities)),
	}

	for _, entity := range entities {
		started := time.Now()
		rows, err := a.syncEntity(ctx, entity, from, to)
		item := models.ETLEntityResult{
			Entity:        entity,
			Rows:          rows,
			ExecutionTime: time.Since(started).Milliseconds(),
			Success:       err == nil,
		}
		if err != nil {
			item.Error = err.Error()
			slog.Error("etl entity failed", "entity", entity, "error", err)
		} else if err := a.syncLog.Upsert(ctx, entity, to); err != nil {
			item.Success = false
			item.Error = err.Error()
			slog.Error("etl sync log failed", "entity", entity, "error", err)
		}
		result.Results = append(result.Results, item)
	}

	ok := 0
	for _, r := range result.Results {
		if r.Success {
			ok++
		}
	}
	result.Message = fmt.Sprintf("%d/%d entities synced", ok, len(result.Results))
	return result, nil
}

// SyncOrderByID re-sincroniza una sola orden contra OLAP, junto con todo su
// historial de status_history (alimentado por trigger en OLTP, no requiere
// evento propio). La llama el consumidor de eventos (cmd/etl) al recibir
// order_status_changed.
func (a *etlApp) SyncOrderByID(ctx context.Context, orderID uuid.UUID) error {
	row, err := a.source.FetchOrderByID(ctx, orderID)
	if err != nil {
		return err
	}
	row.GMV = models.GMV(row.TotalPrice, row.DeliveryFee)
	if err := a.sink.UpsertOrders(ctx, []models.CubeOrder{*row}); err != nil {
		return err
	}

	history, err := a.source.FetchOrderStatusChangesByOrderID(ctx, orderID)
	if err != nil {
		return err
	}
	return a.sink.UpsertOrderStatusChanges(ctx, history)
}

// SyncFareQuoteByID re-sincroniza una sola cotización contra OLAP. La llama
// el consumidor de eventos (cmd/etl) al recibir fare_quote_created.
func (a *etlApp) SyncFareQuoteByID(ctx context.Context, quoteID uuid.UUID) error {
	row, err := a.source.FetchFareQuoteByID(ctx, quoteID)
	if err != nil {
		return err
	}
	return a.sink.UpsertFareQuotes(ctx, []models.CubeFareQuote{*row})
}

// SyncDeliveryQuoteByID re-sincroniza una sola cotización de delivery contra
// OLAP. La llama el consumidor de eventos (cmd/etl) al recibir delivery_quote_created.
func (a *etlApp) SyncDeliveryQuoteByID(ctx context.Context, quoteID uuid.UUID) error {
	row, err := a.source.FetchDeliveryQuoteByID(ctx, quoteID)
	if err != nil {
		return err
	}
	return a.sink.UpsertDeliveryQuotes(ctx, []models.CubeDeliveryQuote{*row})
}

// SyncTripByID re-sincroniza un solo viaje contra OLAP. La llama el
// consumidor de eventos (cmd/etl) al recibir trip_status_changed.
func (a *etlApp) SyncTripByID(ctx context.Context, tripID uuid.UUID) error {
	row, err := a.source.FetchTripByID(ctx, tripID)
	if err != nil {
		return err
	}
	return a.sink.UpsertTrips(ctx, []models.CubeTrip{*row})
}

// SyncShopByID re-sincroniza un solo local contra OLAP. La llama el
// consumidor de eventos (cmd/etl) al recibir shop_updated.
func (a *etlApp) SyncShopByID(ctx context.Context, shopID uuid.UUID) error {
	row, err := a.source.FetchShopByID(ctx, shopID)
	if err != nil {
		return err
	}
	return a.sink.UpsertShops(ctx, []models.DimShop{*row})
}

// SyncRiderByID re-sincroniza un solo conductor contra OLAP. La llama el
// consumidor de eventos (cmd/etl) al recibir rider_updated.
func (a *etlApp) SyncRiderByID(ctx context.Context, riderID uuid.UUID) error {
	row, err := a.source.FetchRiderByID(ctx, riderID)
	if err != nil {
		return err
	}
	return a.sink.UpsertRiders(ctx, []models.DimRider{*row})
}

// SyncUserByID re-sincroniza una sola cuenta contra OLAP. La llama el
// consumidor de eventos (cmd/etl) al recibir user_updated.
func (a *etlApp) SyncUserByID(ctx context.Context, userID uuid.UUID) error {
	row, err := a.source.FetchUserByID(ctx, userID)
	if err != nil {
		return err
	}
	return a.sink.UpsertUsers(ctx, []models.DimUser{*row})
}

func (a *etlApp) syncEntity(ctx context.Context, entity models.ETLEntity, from, to time.Time) (int, error) {
	switch entity {
	case models.EntityDimShop:
		rows, err := a.source.FetchShops(ctx, from, to)
		if err != nil {
			return 0, err
		}
		return len(rows), a.sink.UpsertShops(ctx, rows)
	case models.EntityDimUser:
		rows, err := a.source.FetchUsers(ctx, from, to)
		if err != nil {
			return 0, err
		}
		return len(rows), a.sink.UpsertUsers(ctx, rows)
	case models.EntityDimRider:
		rows, err := a.source.FetchRiders(ctx, from, to)
		if err != nil {
			return 0, err
		}
		return len(rows), a.sink.UpsertRiders(ctx, rows)
	case models.EntityCubeOrder:
		rows, err := a.source.FetchOrders(ctx, from, to)
		if err != nil {
			return 0, err
		}
		for i := range rows {
			rows[i].GMV = models.GMV(rows[i].TotalPrice, rows[i].DeliveryFee)
		}
		return len(rows), a.sink.UpsertOrders(ctx, rows)
	case models.EntityCubeTrip:
		rows, err := a.source.FetchTrips(ctx, from, to)
		if err != nil {
			return 0, err
		}
		return len(rows), a.sink.UpsertTrips(ctx, rows)
	case models.EntityCubeFareQuote:
		rows, err := a.source.FetchFareQuotes(ctx, from, to)
		if err != nil {
			return 0, err
		}
		return len(rows), a.sink.UpsertFareQuotes(ctx, rows)
	case models.EntityCubeDeliveryQuote:
		rows, err := a.source.FetchDeliveryQuotes(ctx, from, to)
		if err != nil {
			return 0, err
		}
		return len(rows), a.sink.UpsertDeliveryQuotes(ctx, rows)
	case models.EntityFactTrackingEvent:
		rows, err := a.source.FetchTrackingEvents(ctx, from, to)
		if err != nil {
			return 0, err
		}
		return len(rows), a.sink.UpsertTrackingEvents(ctx, rows)
	case models.EntityFactOrderStatusChange:
		rows, err := a.source.FetchOrderStatusChanges(ctx, from, to)
		if err != nil {
			return 0, err
		}
		return len(rows), a.sink.UpsertOrderStatusChanges(ctx, rows)
	default:
		return 0, errcustom.ErrUnknownETLEntity
	}
}
