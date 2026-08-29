package repo

import (
	"context"

	"Encargalo.app-api.go/internal/analytics/domain/models"
	"Encargalo.app-api.go/internal/analytics/domain/ports"
	"Encargalo.app-api.go/internal/shared/adapters/postgres"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/uptrace/bun"
)

type olapSinkRepo struct {
	db *bun.DB
}

func NewOLAPSinkRepo(olap *postgres.OLAPDB) ports.OLAPSinkRepo {
	return &olapSinkRepo{db: olap.DB}
}

func (r *olapSinkRepo) UpsertShops(ctx context.Context, rows []models.DimShop) error {
	return insertInChunks(rows, func(chunk []models.DimShop) error {
		_, err := r.db.NewInsert().
			Model(&chunk).
			On("CONFLICT (shop_id) DO UPDATE").
			Set("name = EXCLUDED.name").
			Set("tag = EXCLUDED.tag").
			Set("address = EXCLUDED.address").
			Set("type = EXCLUDED.type").
			Set("license_status = EXCLUDED.license_status").
			Set("opened = EXCLUDED.opened").
			Set("created_at = EXCLUDED.created_at").
			Set("updated_at = EXCLUDED.updated_at").
			Set("synced_at = EXCLUDED.synced_at").
			Set("h3_cell = EXCLUDED.h3_cell").
			Exec(ctx)
		if err != nil {
			return errcustom.Unexpected(err)
		}
		return nil
	})
}

func (r *olapSinkRepo) UpsertUsers(ctx context.Context, rows []models.DimUser) error {
	return insertInChunks(rows, func(chunk []models.DimUser) error {
		_, err := r.db.NewInsert().
			Model(&chunk).
			On("CONFLICT (user_id) DO UPDATE").
			Set("name = EXCLUDED.name").
			Set("sur_name = EXCLUDED.sur_name").
			Set("phone = EXCLUDED.phone").
			Set("email = EXCLUDED.email").
			Set("roles = EXCLUDED.roles").
			Set("enabled = EXCLUDED.enabled").
			Set("registered_at = EXCLUDED.registered_at").
			Set("updated_at = EXCLUDED.updated_at").
			Set("synced_at = EXCLUDED.synced_at").
			Exec(ctx)
		if err != nil {
			return errcustom.Unexpected(err)
		}
		return nil
	})
}

func (r *olapSinkRepo) UpsertRiders(ctx context.Context, rows []models.DimRider) error {
	return insertInChunks(rows, func(chunk []models.DimRider) error {
		_, err := r.db.NewInsert().
			Model(&chunk).
			On("CONFLICT (rider_id) DO UPDATE").
			Set("name = EXCLUDED.name").
			Set("sur_name = EXCLUDED.sur_name").
			Set("phone = EXCLUDED.phone").
			Set("email = EXCLUDED.email").
			Set("status = EXCLUDED.status").
			Set("vehicle_type = EXCLUDED.vehicle_type").
			Set("registered_at = EXCLUDED.registered_at").
			Set("activated_at = EXCLUDED.activated_at").
			Set("updated_at = EXCLUDED.updated_at").
			Set("synced_at = EXCLUDED.synced_at").
			Exec(ctx)
		if err != nil {
			return errcustom.Unexpected(err)
		}
		return nil
	})
}

func (r *olapSinkRepo) UpsertOrders(ctx context.Context, rows []models.CubeOrder) error {
	return insertInChunks(rows, func(chunk []models.CubeOrder) error {
		_, err := r.db.NewInsert().
			Model(&chunk).
			On("CONFLICT (order_id) DO UPDATE").
			Set("shop_id = EXCLUDED.shop_id").
			Set("customer_id = EXCLUDED.customer_id").
			Set("rider_id = EXCLUDED.rider_id").
			Set("status = EXCLUDED.status").
			Set("method_payment = EXCLUDED.method_payment").
			Set("total_price = EXCLUDED.total_price").
			Set("delivery_fee = EXCLUDED.delivery_fee").
			Set("gmv = EXCLUDED.gmv").
			Set("promo_subsidy_amount = EXCLUDED.promo_subsidy_amount").
			Set("is_launch_promo = EXCLUDED.is_launch_promo").
			Set("is_zero_commission = EXCLUDED.is_zero_commission").
			Set("order_date = EXCLUDED.order_date").
			Set("created_at = EXCLUDED.created_at").
			Set("synced_at = EXCLUDED.synced_at").
			Set("h3_cell = EXCLUDED.h3_cell").
			Exec(ctx)
		if err != nil {
			return errcustom.Unexpected(err)
		}
		return nil
	})
}

func (r *olapSinkRepo) UpsertTrips(ctx context.Context, rows []models.CubeTrip) error {
	return insertInChunks(rows, func(chunk []models.CubeTrip) error {
		_, err := r.db.NewInsert().
			Model(&chunk).
			On("CONFLICT (trip_id) DO UPDATE").
			Set("customer_id = EXCLUDED.customer_id").
			Set("rider_id = EXCLUDED.rider_id").
			Set("status = EXCLUDED.status").
			Set("fare = EXCLUDED.fare").
			Set("pickup_h3_cell = EXCLUDED.pickup_h3_cell").
			Set("dropoff_h3_cell = EXCLUDED.dropoff_h3_cell").
			Set("trip_date = EXCLUDED.trip_date").
			Set("created_at = EXCLUDED.created_at").
			Set("synced_at = EXCLUDED.synced_at").
			Exec(ctx)
		if err != nil {
			return errcustom.Unexpected(err)
		}
		return nil
	})
}

func (r *olapSinkRepo) UpsertTrackingEvents(ctx context.Context, rows []models.FactTrackingEvent) error {
	return insertInChunks(rows, func(chunk []models.FactTrackingEvent) error {
		_, err := r.db.NewInsert().
			Model(&chunk).
			On("CONFLICT (id) DO UPDATE").
			Set("type = EXCLUDED.type").
			Set("params = EXCLUDED.params").
			Set("created_at = EXCLUDED.created_at").
			Set("synced_at = EXCLUDED.synced_at").
			Exec(ctx)
		if err != nil {
			return errcustom.Unexpected(err)
		}
		return nil
	})
}
