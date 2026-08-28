package ports

import (
	"context"

	"Encargalo.app-api.go/internal/trips/domain/dtos"
	"Encargalo.app-api.go/internal/trips/domain/models"
	"github.com/google/uuid"
)

type TripsApp interface {
	RequestTrip(ctx context.Context, customerID uuid.UUID, req dtos.CreateTripRequest) (dtos.TripResponse, error)
	GetTrip(ctx context.Context, tripID, requesterID uuid.UUID) (dtos.TripResponse, error)
	GetTripHistory(ctx context.Context, customerID uuid.UUID) (dtos.TripHistoryResponse, error)
	CancelTrip(ctx context.Context, tripID, requesterID uuid.UUID) (dtos.TripResponse, error)
}

type TripsRepo interface {
	CreateTrip(ctx context.Context, trip *models.Trip) error
	GetTripByID(ctx context.Context, tripID uuid.UUID) (*models.Trip, error)
	GetTripHistoryByCustomer(ctx context.Context, customerID uuid.UUID) ([]models.Trip, error)
	// UpdateTripStatus hace un UPDATE condicional (WHERE status = fromStatus) y
	// setea el timestamp correspondiente (accepted_at/completed_at/cancelled_at)
	// según el nuevo estado. RowsAffected == 0 => el estado ya cambió (CAS).
	UpdateTripStatus(ctx context.Context, tripID uuid.UUID, fromStatus, toStatus models.TripStatus) (bool, error)
}
