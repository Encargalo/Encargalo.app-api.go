package repo

import (
	"context"
	"time"

	"Encargalo.app-api.go/internal/analytics/domain/models"
	"Encargalo.app-api.go/internal/analytics/domain/ports"
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

func (r *oltpSourceRepo) FetchOrders(ctx context.Context, from, to time.Time) ([]models.CubeOrder, error) {
	// Sin updated_at en orders: re-sincronizamos por ventana de created_at (lookback cubre status tardíos).
	var rows []orderRow
	err := r.db.NewSelect().
		TableExpr("orders.orders AS o").
		Column(
			"o.id", "o.shop_id", "o.customer_id", "o.rider_id", "o.status", "o.method_payment",
			"o.total_price", "o.delivery_fee", "o.promo_subsidy_amount", "o.is_launch_promo",
			"o.is_zero_commission", "o.created_at", "o.h3_cell",
		).
		Where("o.created_at >= ? AND o.created_at < ?", from, to).
		Scan(ctx, &rows)
	if err != nil {
		return nil, errcustom.Unexpected(err)
	}

	now := time.Now().UTC()
	out := make([]models.CubeOrder, len(rows))
	for i, row := range rows {
		out[i] = models.CubeOrder{
			OrderID:            row.ID,
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
			SyncedAt:           now,
			H3Cell:             row.H3Cell,
		}
	}
	return out, nil
}

type tripRow struct {
	ID                uuid.UUID  `bun:"id"`
	CustomerID        uuid.UUID  `bun:"customer_id"`
	RiderID           *uuid.UUID `bun:"rider_id"`
	Status            string     `bun:"status"`
	Fare              *float64   `bun:"fare"`
	PickupH3CellZone  *string    `bun:"pickup_h3_cell_zone"`
	DropoffH3CellZone *string    `bun:"dropoff_h3_cell_zone"`
	CreatedAt         time.Time  `bun:"created_at"`
}

// Sin updated_at en trips: re-sincronizamos por ventana de created_at (lookback cubre status tardíos).
func (r *oltpSourceRepo) FetchTrips(ctx context.Context, from, to time.Time) ([]models.CubeTrip, error) {
	var rows []tripRow
	err := r.db.NewSelect().
		TableExpr("trips.trips AS t").
		Column(
			"t.id", "t.customer_id", "t.rider_id", "t.status", "t.fare",
			"t.pickup_h3_cell_zone", "t.dropoff_h3_cell_zone", "t.created_at",
		).
		Where("t.created_at >= ? AND t.created_at < ?", from, to).
		Scan(ctx, &rows)
	if err != nil {
		return nil, errcustom.Unexpected(err)
	}

	now := time.Now().UTC()
	out := make([]models.CubeTrip, len(rows))
	for i, row := range rows {
		out[i] = models.CubeTrip{
			TripID:        row.ID,
			CustomerID:    row.CustomerID,
			RiderID:       row.RiderID,
			Status:        row.Status,
			Fare:          row.Fare,
			PickupH3Cell:  row.PickupH3CellZone,
			DropoffH3Cell: row.DropoffH3CellZone,
			TripDate:      row.CreatedAt.UTC().Truncate(24 * time.Hour),
			CreatedAt:     row.CreatedAt,
			SyncedAt:      now,
		}
	}
	return out, nil
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
