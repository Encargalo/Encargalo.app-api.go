package repo

import (
	"context"
	"log/slog"
	"time"

	"Encargalo.app-api.go/internal/pkg/geo"
	"Encargalo.app-api.go/internal/riders/domain/models"
	"Encargalo.app-api.go/internal/riders/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type lastKnownLocationRepo struct {
	db *bun.DB
}

func NewLastKnownLocationRepo(db *bun.DB) ports.LastKnownLocationStore {
	return &lastKnownLocationRepo{db: db}
}

func (r *lastKnownLocationRepo) Save(ctx context.Context, loc models.RiderLastKnownLocation) error {
	loc.UpdatedAt = time.Now()
	_, err := r.db.NewInsert().
		Model(&loc).
		On("CONFLICT (rider_id) DO UPDATE").
		Set("latitude = EXCLUDED.latitude").
		Set("longitude = EXCLUDED.longitude").
		Set("mode = EXCLUDED.mode").
		Set("vehicle = EXCLUDED.vehicle").
		Set("updated_at = EXCLUDED.updated_at").
		Exec(ctx)
	if err != nil {
		slog.Error("riderLastKnownLocation Save failed", "riderID", loc.RiderID, "error", err)
		return errcustom.Unexpected(err)
	}
	return nil
}

// SearchNearby pre-filtra con un bounding box barato en SQL (sin PostGIS) y
// afina la distancia exacta con Haversine en Go — es un respaldo, no el
// matching en vivo, así que no necesita un índice espacial.
func (r *lastKnownLocationRepo) SearchNearby(ctx context.Context, query models.LastKnownLocationQuery) ([]models.RiderLastKnownLocation, error) {
	latDelta, lngDelta := geo.BoundingBoxDeltas(query.Latitude, query.RadiusKm)

	q := r.db.NewSelect().
		Model((*models.RiderLastKnownLocation)(nil)).
		Where("mode = ?", query.Mode).
		Where("updated_at >= ?", time.Now().Add(-query.MaxAge)).
		Where("latitude BETWEEN ? AND ?", query.Latitude-latDelta, query.Latitude+latDelta).
		Where("longitude BETWEEN ? AND ?", query.Longitude-lngDelta, query.Longitude+lngDelta)

	if len(query.ExcludeVehicles) > 0 {
		q = q.Where("vehicle NOT IN (?)", bun.In(query.ExcludeVehicles))
	}
	if len(query.Exclude) > 0 {
		excluded := make([]uuid.UUID, 0, len(query.Exclude))
		for id := range query.Exclude {
			excluded = append(excluded, id)
		}
		q = q.Where("rider_id NOT IN (?)", bun.In(excluded))
	}

	var rows []models.RiderLastKnownLocation
	if err := q.Scan(ctx, &rows); err != nil {
		slog.Error("riderLastKnownLocation SearchNearby failed", "error", err)
		return nil, errcustom.Unexpected(err)
	}

	out := make([]models.RiderLastKnownLocation, 0, len(rows))
	for _, row := range rows {
		if geo.HaversineKm(query.Latitude, query.Longitude, row.Latitude, row.Longitude) <= query.RadiusKm {
			out = append(out, row)
		}
	}
	return out, nil
}
