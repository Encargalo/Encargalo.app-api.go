package repo

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"Encargalo.app-api.go/internal/analytics/domain/models"
	"Encargalo.app-api.go/internal/analytics/domain/ports"
	"Encargalo.app-api.go/internal/pkg/bizclock"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type oltpSourceRepo struct {
	db *bun.DB
}

func NewOLTPSourceRepo(db *bun.DB) ports.OLTPSourceRepo {
	return &oltpSourceRepo{db: db}
}

type shopRow struct {
	ID            uuid.UUID `bun:"id"`
	Name          string    `bun:"name"`
	Slug          string    `bun:"slug"`
	Address       string    `bun:"address"`
	Type          string    `bun:"type"`
	LicenseStatus string    `bun:"license_status"`
	Opened        bool      `bun:"opened"`
	CreatedAt     time.Time `bun:"created_at"`
	UpdatedAt     time.Time `bun:"updated_at"`
	H3Cell        *string   `bun:"h3_cell"`
}

func (r *oltpSourceRepo) FetchShops(ctx context.Context, from, to time.Time) ([]models.DimShop, error) {
	var rows []shopRow
	err := r.db.NewSelect().
		TableExpr("business.shops AS s").
		Column("s.id", "s.name", "s.slug", "s.address", "s.type", "s.license_status", "s.opened", "s.created_at", "s.updated_at", "s.h3_cell").
		Where("s.deleted_at IS NULL").
		Where("s.updated_at >= ? AND s.updated_at < ?", from, to).
		Scan(ctx, &rows)
	if err != nil {
		return nil, errcustom.Unexpected(err)
	}

	now := time.Now().UTC()
	out := make([]models.DimShop, len(rows))
	for i, row := range rows {
		out[i] = models.DimShop{
			ShopID:        row.ID,
			Name:          row.Name,
			Slug:          row.Slug,
			Address:       row.Address,
			Type:          row.Type,
			LicenseStatus: row.LicenseStatus,
			Opened:        row.Opened,
			CreatedAt:     row.CreatedAt,
			UpdatedAt:     row.UpdatedAt,
			SyncedAt:      now,
			H3Cell:        row.H3Cell,
		}
	}
	return out, nil
}

type userRow struct {
	ID        uuid.UUID `bun:"id"`
	Name      string    `bun:"name"`
	SurName   *string   `bun:"sur_name"`
	Phone     *string   `bun:"phone"`
	Email     *string   `bun:"email"`
	Enabled   bool      `bun:"enabled"`
	CreatedAt time.Time `bun:"created_at"`
	UpdatedAt time.Time `bun:"updated_at"`
	Roles     []string  `bun:"roles,array"`
}

func (r *oltpSourceRepo) FetchUsers(ctx context.Context, from, to time.Time) ([]models.DimUser, error) {
	var rows []userRow
	err := r.db.NewSelect().
		TableExpr("users.accounts AS a").
		Column("a.id", "a.name", "a.sur_name", "a.phone", "a.email", "a.enabled", "a.created_at", "a.updated_at").
		ColumnExpr("COALESCE(array_agg(DISTINCT r.name) FILTER (WHERE r.name IS NOT NULL), '{}') AS roles").
		Join("LEFT JOIN users.user_roles ur ON ur.user_id = a.id AND ur.enabled = TRUE").
		Join("LEFT JOIN users.roles r ON r.id = ur.role_id").
		Where("a.deleted_at IS NULL").
		Where("a.updated_at >= ? AND a.updated_at < ?", from, to).
		GroupExpr("a.id, a.name, a.sur_name, a.phone, a.email, a.enabled, a.created_at, a.updated_at").
		Scan(ctx, &rows)
	if err != nil {
		return nil, errcustom.Unexpected(err)
	}

	now := time.Now().UTC()
	out := make([]models.DimUser, len(rows))
	for i, row := range rows {
		roles := row.Roles
		if roles == nil {
			roles = []string{}
		}
		out[i] = models.DimUser{
			UserID:       row.ID,
			Name:         row.Name,
			SurName:      row.SurName,
			Phone:        row.Phone,
			Email:        row.Email,
			Roles:        roles,
			Enabled:      row.Enabled,
			RegisteredAt: row.CreatedAt,
			UpdatedAt:    row.UpdatedAt,
			SyncedAt:     now,
		}
	}
	return out, nil
}

type riderRow struct {
	UserID      uuid.UUID  `bun:"user_id"`
	Name        string     `bun:"name"`
	SurName     *string    `bun:"sur_name"`
	Phone       *string    `bun:"phone"`
	Email       *string    `bun:"email"`
	Status      string     `bun:"status"`
	VehicleType *string    `bun:"vehicle_type"`
	CreatedAt   time.Time  `bun:"created_at"`
	ActivatedAt *time.Time `bun:"activated_at"`
	UpdatedAt   time.Time  `bun:"updated_at"`
}

func (r *oltpSourceRepo) FetchRiders(ctx context.Context, from, to time.Time) ([]models.DimRider, error) {
	var rows []riderRow
	err := r.db.NewSelect().
		TableExpr("riders.profiles AS p").
		Column("p.user_id", "p.status", "p.vehicle_type", "p.created_at", "p.activated_at", "p.updated_at").
		ColumnExpr("a.name AS name").
		ColumnExpr("a.sur_name AS sur_name").
		ColumnExpr("a.phone AS phone").
		ColumnExpr("a.email AS email").
		Join("JOIN users.accounts AS a ON a.id = p.user_id").
		Where("p.updated_at >= ? AND p.updated_at < ?", from, to).
		Scan(ctx, &rows)
	if err != nil {
		return nil, errcustom.Unexpected(err)
	}

	now := time.Now().UTC()
	out := make([]models.DimRider, len(rows))
	for i, row := range rows {
		out[i] = models.DimRider{
			RiderID:      row.UserID,
			Name:         row.Name,
			SurName:      row.SurName,
			Phone:        row.Phone,
			Email:        row.Email,
			Status:       row.Status,
			VehicleType:  row.VehicleType,
			RegisteredAt: row.CreatedAt,
			ActivatedAt:  row.ActivatedAt,
			UpdatedAt:    row.UpdatedAt,
			SyncedAt:     now,
		}
	}
	return out, nil
}

type orderRow struct {
	ID                 uuid.UUID  `bun:"id"`
	QuoteID            *uuid.UUID `bun:"quote_id"`
	ShopID             uuid.UUID  `bun:"shop_id"`
	CustomerID         uuid.UUID  `bun:"customer_id"`
	RiderID            *uuid.UUID `bun:"rider_id"`
	Status             string     `bun:"status"`
	MethodPayment      string     `bun:"method_payment"`
	TotalPrice         float64    `bun:"total_price"`
	DeliveryFee        float64    `bun:"delivery_fee"`
	PromoSubsidyAmount float64    `bun:"promo_subsidy_amount"`
	IsLaunchPromo      bool       `bun:"is_launch_promo"`
	IsZeroCommission   bool       `bun:"is_zero_commission"`
	CreatedAt          time.Time  `bun:"created_at"`
	H3Cell             *string    `bun:"h3_cell"`
}

var orderColumns = []string{
	"o.id", "o.quote_id", "o.shop_id", "o.customer_id", "o.rider_id", "o.status", "o.method_payment",
	"o.total_price", "o.delivery_fee", "o.promo_subsidy_amount", "o.is_launch_promo",
	"o.is_zero_commission", "o.created_at", "o.h3_cell",
}

func orderRowToCubeOrder(row orderRow, syncedAt time.Time) models.CubeOrder {
	return models.CubeOrder{
		OrderID:            row.ID,
		QuoteID:            row.QuoteID,
		ShopID:             row.ShopID,
		CustomerID:         row.CustomerID,
		RiderID:            row.RiderID,
		Status:             row.Status,
		MethodPayment:      row.MethodPayment,
		TotalPrice:         row.TotalPrice,
		DeliveryFee:        row.DeliveryFee,
		PromoSubsidyAmount: row.PromoSubsidyAmount,
		IsLaunchPromo:      row.IsLaunchPromo,
		IsZeroCommission:   row.IsZeroCommission,
		OrderDate:          row.CreatedAt.UTC().Truncate(24 * time.Hour),
		CreatedAt:          row.CreatedAt,
		SyncedAt:           syncedAt,
		H3Cell:             row.H3Cell,
	}
}

func (r *oltpSourceRepo) FetchOrders(ctx context.Context, from, to time.Time) ([]models.CubeOrder, error) {
	// Sin updated_at en orders: re-sincronizamos por ventana de created_at (lookback cubre status tardíos).
	var rows []orderRow
	err := r.db.NewSelect().
		TableExpr("orders.orders AS o").
		Column(orderColumns...).
		Where("o.created_at >= ? AND o.created_at < ?", from, to).
		Scan(ctx, &rows)
	if err != nil {
		return nil, errcustom.Unexpected(err)
	}

	now := time.Now().UTC()
	out := make([]models.CubeOrder, len(rows))
	for i, row := range rows {
		out[i] = orderRowToCubeOrder(row, now)
	}
	return out, nil
}

// FetchOrderByID resincroniza una sola orden puntualmente: lo usa el
// consumidor de eventos para reaccionar a order_status_changed sin esperar
// a que la orden caiga en la ventana del próximo batch por fecha.
func (r *oltpSourceRepo) FetchOrderByID(ctx context.Context, orderID uuid.UUID) (*models.CubeOrder, error) {
	var row orderRow
	err := r.db.NewSelect().
		TableExpr("orders.orders AS o").
		Column(orderColumns...).
		Where("o.id = ?", orderID).
		Scan(ctx, &row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errcustom.ErrNotFound
		}
		return nil, errcustom.Unexpected(err)
	}

	cube := orderRowToCubeOrder(row, time.Now().UTC())
	return &cube, nil
}

type tripRow struct {
	ID                uuid.UUID  `bun:"id"`
	QuoteID           *uuid.UUID `bun:"quote_id"`
	CustomerID        uuid.UUID  `bun:"customer_id"`
	RiderID           *uuid.UUID `bun:"rider_id"`
	Status            string     `bun:"status"`
	Fare              *float64   `bun:"fare"`
	PickupH3CellZone  *string    `bun:"pickup_h3_cell_zone"`
	DropoffH3CellZone *string    `bun:"dropoff_h3_cell_zone"`
	CreatedAt         time.Time  `bun:"created_at"`
}

var tripColumns = []string{
	"t.id", "t.quote_id", "t.customer_id", "t.rider_id", "t.status", "t.fare",
	"t.pickup_h3_cell_zone", "t.dropoff_h3_cell_zone", "t.created_at",
}

func tripRowToCubeTrip(row tripRow, syncedAt time.Time) models.CubeTrip {
	return models.CubeTrip{
		TripID:        row.ID,
		QuoteID:       row.QuoteID,
		CustomerID:    row.CustomerID,
		RiderID:       row.RiderID,
		Status:        row.Status,
		Fare:          row.Fare,
		PickupH3Cell:  row.PickupH3CellZone,
		DropoffH3Cell: row.DropoffH3CellZone,
		TripDate:      row.CreatedAt.UTC().Truncate(24 * time.Hour),
		CreatedAt:     row.CreatedAt,
		SyncedAt:      syncedAt,
	}
}

// Sin updated_at en trips: re-sincronizamos por ventana de created_at (lookback cubre status tardíos).
func (r *oltpSourceRepo) FetchTrips(ctx context.Context, from, to time.Time) ([]models.CubeTrip, error) {
	var rows []tripRow
	err := r.db.NewSelect().
		TableExpr("trips.trips AS t").
		Column(tripColumns...).
		Where("t.created_at >= ? AND t.created_at < ?", from, to).
		Scan(ctx, &rows)
	if err != nil {
		return nil, errcustom.Unexpected(err)
	}

	now := time.Now().UTC()
	out := make([]models.CubeTrip, len(rows))
	for i, row := range rows {
		out[i] = tripRowToCubeTrip(row, now)
	}
	return out, nil
}

// FetchTripByID resincroniza un solo viaje puntualmente, en reacción a
// trip_status_changed.
func (r *oltpSourceRepo) FetchTripByID(ctx context.Context, tripID uuid.UUID) (*models.CubeTrip, error) {
	var row tripRow
	err := r.db.NewSelect().
		TableExpr("trips.trips AS t").
		Column(tripColumns...).
		Where("t.id = ?", tripID).
		Scan(ctx, &row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errcustom.ErrNotFound
		}
		return nil, errcustom.Unexpected(err)
	}

	cube := tripRowToCubeTrip(row, time.Now().UTC())
	return &cube, nil
}

type trackingEventRow struct {
	ID        uuid.UUID      `bun:"id"`
	Type      string         `bun:"type"`
	Params    map[string]any `bun:"params,type:jsonb"`
	CreatedAt time.Time      `bun:"created_at"`
}

func (r *oltpSourceRepo) FetchTrackingEvents(ctx context.Context, from, to time.Time) ([]models.FactTrackingEvent, error) {
	var rows []trackingEventRow
	err := r.db.NewSelect().
		TableExpr("tracking.events AS e").
		Column("e.id", "e.type", "e.params", "e.created_at").
		Where("e.created_at >= ? AND e.created_at < ?", from, to).
		Scan(ctx, &rows)
	if err != nil {
		return nil, errcustom.Unexpected(err)
	}

	now := time.Now().UTC()
	out := make([]models.FactTrackingEvent, len(rows))
	for i, row := range rows {
		params := row.Params
		if params == nil {
			params = map[string]any{}
		}
		out[i] = models.FactTrackingEvent{
			ID:        row.ID,
			Type:      row.Type,
			Params:    params,
			CreatedAt: row.CreatedAt,
			SyncedAt:  now,
		}
	}
	return out, nil
}

type fareQuoteRow struct {
	QuoteID           uuid.UUID `bun:"quote_id"`
	CustomerID        uuid.UUID `bun:"customer_id"`
	DistanceMeters    float64   `bun:"distance_meters"`
	DurationMin       int64     `bun:"duration_min"`
	FareUSD           float64   `bun:"fare_usd"`
	FareBand          string    `bun:"fare_band"`
	PickupH3CellZone  *string   `bun:"pickup_h3_cell_zone"`
	DropoffH3CellZone *string   `bun:"dropoff_h3_cell_zone"`
	CreatedAt         time.Time `bun:"created_at"`
}

var fareQuoteColumns = []string{
	"q.quote_id", "q.customer_id", "q.distance_meters", "q.duration_min",
	"q.fare_usd", "q.fare_band", "q.pickup_h3_cell_zone",
	"q.dropoff_h3_cell_zone", "q.created_at",
}

func fareQuoteRowToCubeFareQuote(row fareQuoteRow, syncedAt time.Time) models.CubeFareQuote {
	local := row.CreatedAt.In(bizclock.Location())
	return models.CubeFareQuote{
		QuoteID:        row.QuoteID,
		CustomerID:     row.CustomerID,
		DistanceMeters: row.DistanceMeters,
		DurationMin:    row.DurationMin,
		FareUSD:        row.FareUSD,
		FareBand:       row.FareBand,
		PickupH3Cell:   row.PickupH3CellZone,
		DropoffH3Cell:  row.DropoffH3CellZone,
		QuoteDate:      time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.UTC),
		QuoteHour:      int16(local.Hour()),
		CreatedAt:      row.CreatedAt,
		SyncedAt:       syncedAt,
	}
}

// Las cotizaciones son inmutables: una vez escritas no cambian, así que la
// ventana de created_at las cubre sin necesitar lookback por updated_at.
func (r *oltpSourceRepo) FetchFareQuotes(ctx context.Context, from, to time.Time) ([]models.CubeFareQuote, error) {
	var rows []fareQuoteRow
	err := r.db.NewSelect().
		TableExpr("trips.fare_quotes AS q").
		Column(fareQuoteColumns...).
		Where("q.created_at >= ? AND q.created_at < ?", from, to).
		Scan(ctx, &rows)
	if err != nil {
		return nil, errcustom.Unexpected(err)
	}

	now := time.Now().UTC()
	out := make([]models.CubeFareQuote, len(rows))
	for i, row := range rows {
		out[i] = fareQuoteRowToCubeFareQuote(row, now)
	}
	return out, nil
}

// FetchFareQuoteByID resincroniza una sola cotización puntualmente, en
// reacción a fare_quote_created.
func (r *oltpSourceRepo) FetchFareQuoteByID(ctx context.Context, quoteID uuid.UUID) (*models.CubeFareQuote, error) {
	var row fareQuoteRow
	err := r.db.NewSelect().
		TableExpr("trips.fare_quotes AS q").
		Column(fareQuoteColumns...).
		Where("q.quote_id = ?", quoteID).
		Scan(ctx, &row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errcustom.ErrNotFound
		}
		return nil, errcustom.Unexpected(err)
	}

	cube := fareQuoteRowToCubeFareQuote(row, time.Now().UTC())
	return &cube, nil
}

type deliveryQuoteRow struct {
	QuoteID        uuid.UUID `bun:"quote_id"`
	CustomerID     uuid.UUID `bun:"customer_id"`
	ShopID         uuid.UUID `bun:"shop_id"`
	DistanceMeters float64   `bun:"distance_meters"`
	FeeUSD         float64   `bun:"fee_usd"`
	FareBand       string    `bun:"fare_band"`
	CreatedAt      time.Time `bun:"created_at"`
}

var deliveryQuoteColumns = []string{
	"q.quote_id", "q.customer_id", "q.shop_id", "q.distance_meters",
	"q.fee_usd", "q.fare_band", "q.created_at",
}

func deliveryQuoteRowToCubeDeliveryQuote(row deliveryQuoteRow, syncedAt time.Time) models.CubeDeliveryQuote {
	local := row.CreatedAt.In(bizclock.Location())
	return models.CubeDeliveryQuote{
		QuoteID:        row.QuoteID,
		CustomerID:     row.CustomerID,
		ShopID:         row.ShopID,
		DistanceMeters: row.DistanceMeters,
		FeeUSD:         row.FeeUSD,
		FareBand:       row.FareBand,
		QuoteDate:      time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.UTC),
		QuoteHour:      int16(local.Hour()),
		CreatedAt:      row.CreatedAt,
		SyncedAt:       syncedAt,
	}
}

// Las cotizaciones son inmutables: una vez escritas no cambian, así que la
// ventana de created_at las cubre sin necesitar lookback por updated_at.
func (r *oltpSourceRepo) FetchDeliveryQuotes(ctx context.Context, from, to time.Time) ([]models.CubeDeliveryQuote, error) {
	var rows []deliveryQuoteRow
	err := r.db.NewSelect().
		TableExpr("orders.fare_quotes AS q").
		Column(deliveryQuoteColumns...).
		Where("q.created_at >= ? AND q.created_at < ?", from, to).
		Scan(ctx, &rows)
	if err != nil {
		return nil, errcustom.Unexpected(err)
	}

	now := time.Now().UTC()
	out := make([]models.CubeDeliveryQuote, len(rows))
	for i, row := range rows {
		out[i] = deliveryQuoteRowToCubeDeliveryQuote(row, now)
	}
	return out, nil
}

// FetchDeliveryQuoteByID resincroniza una sola cotización puntualmente, en
// reacción a delivery_quote_created.
func (r *oltpSourceRepo) FetchDeliveryQuoteByID(ctx context.Context, quoteID uuid.UUID) (*models.CubeDeliveryQuote, error) {
	var row deliveryQuoteRow
	err := r.db.NewSelect().
		TableExpr("orders.fare_quotes AS q").
		Column(deliveryQuoteColumns...).
		Where("q.quote_id = ?", quoteID).
		Scan(ctx, &row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errcustom.ErrNotFound
		}
		return nil, errcustom.Unexpected(err)
	}

	cube := deliveryQuoteRowToCubeDeliveryQuote(row, time.Now().UTC())
	return &cube, nil
}
