package ports

import (
	"context"
	"time"

	"Encargalo.app-api.go/internal/trips/domain/dtos"
	"Encargalo.app-api.go/internal/trips/domain/models"
	"github.com/google/uuid"
)

type TripsApp interface {
	RequestTrip(ctx context.Context, customerID uuid.UUID, req dtos.CreateTripRequest) (dtos.TripResponse, error)
	GetTrip(ctx context.Context, tripID, requesterID uuid.UUID) (dtos.TripResponse, error)
	GetTripHistory(ctx context.Context, customerID uuid.UUID) (dtos.TripHistoryResponse, error)
	CancelTrip(ctx context.Context, tripID, requesterID uuid.UUID) (dtos.TripResponse, error)
	// EstimateFare congela el precio en una TripQuote; RequestTrip la exige en vez de recalcular.
	EstimateFare(ctx context.Context, req dtos.FareEstimateRequest) (dtos.FareEstimateResponse, error)
}

type TripsRepo interface {
	CreateTrip(ctx context.Context, trip *models.Trip) error
	GetTripByID(ctx context.Context, tripID uuid.UUID) (*models.Trip, error)
	GetTripHistoryByCustomer(ctx context.Context, customerID uuid.UUID) ([]models.Trip, error)
	UpdateTripStatus(ctx context.Context, tripID uuid.UUID, fromStatus, toStatus models.TripStatus) (bool, error)
	AssignTripRider(ctx context.Context, tripID, riderID uuid.UUID) (bool, error)
	// CompleteTrip hace un UPDATE condicional (WHERE status = in_progress) que
	// setea status = completed, fare y completed_at en un solo movimiento
	// atómico. RowsAffected == 0 => el estado ya cambió (CAS).
	CompleteTrip(ctx context.Context, tripID uuid.UUID, fare float64) (bool, error)
}

type TripRiderMatcher interface {
	OfferTrip(ctx context.Context, trip models.Trip)
}

// RiderPublicInfoLookup resuelve los datos públicos del rider asignado.
// Lo implementa riders a partir de su perfil y de la cuenta, sin documentos KYC.
type RiderPublicInfoLookup interface {
	GetByRiderID(ctx context.Context, riderID uuid.UUID) (dtos.RiderPublicInfo, error)
}

// TripQuotesRepo persiste TripQuote en Redis con TTL.
type TripQuotesRepo interface {
	Save(ctx context.Context, quote *models.TripQuote, ttl time.Duration) error
	// GetAndConsume borra la cotización atómicamente al leerla (GETDEL); escopada por
	// customerID, así un quote_id ajeno nunca matchea (ni se borra) la de otro cliente.
	GetAndConsume(ctx context.Context, customerID, quoteID uuid.UUID) (*models.TripQuote, error)
}
