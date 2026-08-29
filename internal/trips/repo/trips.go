package repo

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/trips/domain/models"
	"Encargalo.app-api.go/internal/trips/domain/ports"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type tripsRepo struct {
	db *bun.DB
}

func NewTripsRepo(db *bun.DB) ports.TripsRepo {
	return &tripsRepo{db: db}
}

func (r *tripsRepo) CreateTrip(ctx context.Context, trip *models.Trip) error {
	if _, err := r.db.NewInsert().Model(trip).Exec(ctx); err != nil {
		slog.Error("trips CreateTrip failed", "customerID", trip.CustomerID, "error", err)
		return errcustom.Unexpected(err)
	}
	return nil
}

func (r *tripsRepo) GetTripByID(ctx context.Context, tripID uuid.UUID) (*models.Trip, error) {
	var trip models.Trip
	err := r.db.NewSelect().
		Model(&trip).
		Where("id = ?", tripID).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errcustom.ErrNotFound
		}
		slog.Error("trips GetTripByID failed", "tripID", tripID, "error", err)
		return nil, errcustom.Unexpected(err)
	}
	return &trip, nil
}

func (r *tripsRepo) GetTripHistoryByCustomer(ctx context.Context, customerID uuid.UUID) ([]models.Trip, error) {
	var trips []models.Trip
	err := r.db.NewSelect().
		Model(&trips).
		Where("customer_id = ?", customerID).
		Order("created_at DESC").
		Scan(ctx)
	if err != nil {
		slog.Error("trips GetTripHistoryByCustomer failed", "customerID", customerID, "error", err)
		return nil, errcustom.Unexpected(err)
	}
	return trips, nil
}

func (r *tripsRepo) UpdateTripStatus(ctx context.Context, tripID uuid.UUID, fromStatus, toStatus models.TripStatus) (bool, error) {
	q := r.db.NewUpdate().
		TableExpr("trips.trips").
		Set("status = ?", toStatus).
		Set("updated_at = now()")

	switch toStatus {
	case models.TripStatusAccepted:
		q = q.Set("accepted_at = now()")
	case models.TripStatusCompleted:
		q = q.Set("completed_at = now()")
	case models.TripStatusCancelled:
		q = q.Set("cancelled_at = now()")
	}

	res, err := q.
		Where("id = ? AND status = ?", tripID, fromStatus).
		Exec(ctx)
	if err != nil {
		slog.Error("trips UpdateTripStatus failed", "tripID", tripID, "fromStatus", fromStatus, "toStatus", toStatus, "error", err)
		return false, errcustom.Unexpected(err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		slog.Error("trips UpdateTripStatus: failed to read rows affected", "tripID", tripID, "error", err)
		return false, errcustom.Unexpected(err)
	}
	return rows > 0, nil
}

func (r *tripsRepo) AssignTripRider(ctx context.Context, tripID, riderID uuid.UUID) (bool, error) {
	res, err := r.db.NewUpdate().
		TableExpr("trips.trips").
		Set("rider_id = ?", riderID).
		Set("status = ?", models.TripStatusAccepted).
		Set("accepted_at = now()").
		Set("updated_at = now()").
		Where("id = ? AND status = ?", tripID, models.TripStatusSearching).
		Exec(ctx)
	if err != nil {
		slog.Error("trips AssignTripRider failed", "tripID", tripID, "riderID", riderID, "error", err)
		return false, errcustom.Unexpected(err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		slog.Error("trips AssignTripRider: failed to read rows affected", "tripID", tripID, "error", err)
		return false, errcustom.Unexpected(err)
	}
	return rows > 0, nil
}
