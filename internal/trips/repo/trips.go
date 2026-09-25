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

func (r *tripsRepo) SaveTripPayment(ctx context.Context, tripID uuid.UUID, payment *models.TripPayment) error {
	_, err := r.db.NewUpdate().
		TableExpr("trips.trips").
		Set("payment = ?", payment).
		Set("updated_at = now()").
		Where("id = ?", tripID).
		Exec(ctx)
	if err != nil {
		slog.Error("trips SaveTripPayment failed", "tripID", tripID, "error", err)
		return errcustom.Unexpected(err)
	}
	return nil
}

func (r *tripsRepo) ReleaseTripRider(ctx context.Context, tripID, riderID uuid.UUID) (bool, error) {
	res, err := r.db.NewUpdate().
		TableExpr("trips.trips").
		Set("rider_id = NULL").
		Set("status = ?", models.TripStatusSearching).
		Set("accepted_at = NULL").
		Set("pickup_code = NULL").
		Set("updated_at = now()").
		Where("id = ? AND rider_id = ? AND status IN (?)", tripID, riderID, bun.In([]models.TripStatus{
			models.TripStatusAccepted,
			models.TripStatusArriving,
		})).
		Exec(ctx)
	if err != nil {
		slog.Error("trips ReleaseTripRider failed", "tripID", tripID, "riderID", riderID, "error", err)
		return false, errcustom.Unexpected(err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		slog.Error("trips ReleaseTripRider: failed to read rows affected", "tripID", tripID, "error", err)
		return false, errcustom.Unexpected(err)
	}
	return rows > 0, nil
}

func (r *tripsRepo) CompleteTrip(ctx context.Context, tripID uuid.UUID, fare float64, dropoffCode string) (bool, error) {
	res, err := r.db.NewUpdate().
		TableExpr("trips.trips").
		Set("status = ?", models.TripStatusCompleted).
		Set("fare = ?", fare).
		Set("completed_at = now()").
		Set("dropoff_code_used_at = now()").
		Set("updated_at = now()").
		Where("id = ? AND status = ? AND dropoff_code = ? AND dropoff_code_used_at IS NULL", tripID, models.TripStatusInProgress, dropoffCode).
		Exec(ctx)
	if err != nil {
		slog.Error("trips CompleteTrip failed", "tripID", tripID, "error", err)
		return false, errcustom.Unexpected(err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		slog.Error("trips CompleteTrip: failed to read rows affected", "tripID", tripID, "error", err)
		return false, errcustom.Unexpected(err)
	}
	return rows > 0, nil
}

func (r *tripsRepo) StartTripWithPickupCode(ctx context.Context, tripID, riderID uuid.UUID, pickupCode, dropoffCode string) (bool, error) {
	res, err := r.db.NewUpdate().
		TableExpr("trips.trips").
		Set("status = ?", models.TripStatusInProgress).
		Set("pickup_code_used_at = now()").
		Set("dropoff_code = ?", dropoffCode).
		Set("updated_at = now()").
		Where("id = ? AND rider_id = ? AND status = ? AND pickup_code = ? AND pickup_code_used_at IS NULL",
			tripID, riderID, models.TripStatusArriving, pickupCode).
		Exec(ctx)
	if err != nil {
		slog.Error("trips StartTripWithPickupCode failed", "tripID", tripID, "error", err)
		return false, errcustom.Unexpected(err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		slog.Error("trips StartTripWithPickupCode: failed to read rows affected", "tripID", tripID, "error", err)
		return false, errcustom.Unexpected(err)
	}
	return rows > 0, nil
}

func (r *tripsRepo) ListActiveTripsByCustomer(ctx context.Context, customerID uuid.UUID) ([]models.Trip, error) {
	var trips []models.Trip
	err := r.db.NewSelect().
		Model(&trips).
		Where("customer_id = ?", customerID).
		Where("status IN (?)", bun.In([]models.TripStatus{
			models.TripStatusPaymentReview,
			models.TripStatusSearching,
			models.TripStatusAccepted,
			models.TripStatusArriving,
			models.TripStatusInProgress,
		})).
		OrderExpr("requested_at DESC").
		Scan(ctx)
	if err != nil {
		slog.Error("trips ListActiveTripsByCustomer failed", "customerID", customerID, "error", err)
		return nil, errcustom.Unexpected(err)
	}
	return trips, nil
}

func (r *tripsRepo) ListActiveTripsByRider(ctx context.Context, riderID uuid.UUID) ([]models.Trip, error) {
	var trips []models.Trip
	err := r.db.NewSelect().
		Model(&trips).
		Where("rider_id = ?", riderID).
		Where("status IN (?)", bun.In([]models.TripStatus{
			models.TripStatusAccepted,
			models.TripStatusArriving,
			models.TripStatusInProgress,
		})).
		OrderExpr("requested_at DESC").
		Scan(ctx)
	if err != nil {
		slog.Error("trips ListActiveTripsByRider failed", "riderID", riderID, "error", err)
		return nil, errcustom.Unexpected(err)
	}
	return trips, nil
}

func (r *tripsRepo) AssignTripRider(ctx context.Context, tripID, riderID uuid.UUID, pickupCode string) (bool, error) {
	res, err := r.db.NewUpdate().
		TableExpr("trips.trips").
		Set("rider_id = ?", riderID).
		Set("status = ?", models.TripStatusAccepted).
		Set("accepted_at = now()").
		Set("pickup_code = ?", pickupCode).
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
