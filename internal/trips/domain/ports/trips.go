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
	UpdateTripStatus(ctx context.Context, tripID uuid.UUID, fromStatus, toStatus models.TripStatus) (bool, error)
	AssignTripRider(ctx context.Context, tripID, riderID uuid.UUID) (bool, error)
	// AdvanceTripForward intenta accepted->arriving/arriving->in_progress validando el rider_id en la misma query; ok=false no distingue el motivo.
	AdvanceTripForward(ctx context.Context, tripID, riderID uuid.UUID) (trip *models.Trip, ok bool, err error)
	// CompleteTrip hace un UPDATE condicional (WHERE status = in_progress) que
	// setea status = completed, fare y completed_at en un solo movimiento
	// atómico. RowsAffected == 0 => el estado ya cambió (CAS).
	CompleteTrip(ctx context.Context, tripID uuid.UUID, fare float64) (bool, error)
}

type TripRiderMatcher interface {
	OfferTrip(ctx context.Context, trip models.Trip)
}
