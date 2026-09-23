package ports

import (
	"context"

	riderModels "Encargalo.app-api.go/internal/riders/domain/models"
	tripsDtos "Encargalo.app-api.go/internal/trips/domain/dtos"
	tripsModels "Encargalo.app-api.go/internal/trips/domain/models"
	"github.com/google/uuid"
)

// RiderConn abstracts the WebSocket transport so the app layer has no infrastructure dependency.
type RiderConn interface {
	Send(msg []byte)
	Close()
}

// RidersApp defines the operations exposed to the riders HTTP handler.
// Broadcast methods are excluded here; they are accessed via
// orders/domain/ports.OrderRiderBroadcaster by the orders domain.
type RidersApp interface {
	Register(ctx context.Context, riderID uuid.UUID, conn RiderConn, mode riderModels.RiderMode) error
	Unregister(riderID uuid.UUID, conn RiderConn)
	AcceptOrder(ctx context.Context, riderID uuid.UUID, orderID uuid.UUID, conn RiderConn) error
	UpdateLocation(ctx context.Context, riderID uuid.UUID, lat, lng float64) error
	SnapshotTripsRiders() []riderModels.RiderLocationEntry
	AcceptTrip(ctx context.Context, riderID, tripID uuid.UUID, conn RiderConn) error
	// CancelAcceptedTrip libera un viaje accepted o arriving y lo vuelve a ofertar,
	// excluyendo al rider que canceló. in_progress y completed no se pueden cancelar aquí.
	CancelAcceptedTrip(ctx context.Context, riderID, tripID uuid.UUID, conn RiderConn) error
	// AdvanceTrip mueve el trip asignado al siguiente estado de la máquina
	// lineal accepted→arriving→in_progress→completed. Al llegar a completed
	// calcula la tarifa (distancia vial pickup→dropoff) y acredita la
	// ganancia del rider.
	AdvanceTrip(ctx context.Context, riderID, tripID uuid.UUID, code string, conn RiderConn) error
	// SendActiveTrips reenvía al rider que se reconecta en modo trips los viajes
	// que ya tiene asignados (accepted, arriving o in_progress) como trip_status,
	// y vuelve a enlazar su ubicación con el cliente.
	SendActiveTrips(ctx context.Context, riderID uuid.UUID, conn RiderConn) error
	// ActiveTrips devuelve esos mismos viajes por REST, sin códigos de subida/bajada.
	ActiveTrips(ctx context.Context, riderID uuid.UUID) ([]tripsDtos.TripResponse, error)
}

type RidersStream interface {
	PublisherEvent(ctx context.Context, data any) error
}

// TripsStore is the slice of trips persistence the riders domain needs to
// offer and accept trips — narrower than the full trips/domain/ports.TripsRepo
// surface, which also covers trip creation and history unrelated to matching.
type TripsStore interface {
	GetTripByID(ctx context.Context, tripID uuid.UUID) (*tripsModels.Trip, error)
	UpdateTripStatus(ctx context.Context, tripID uuid.UUID, fromStatus, toStatus tripsModels.TripStatus) (bool, error)
	AssignTripRider(ctx context.Context, tripID, riderID uuid.UUID, pickupCode string) (bool, error)
	StartTripWithPickupCode(ctx context.Context, tripID, riderID uuid.UUID, pickupCode, dropoffCode string) (bool, error)
	ReleaseTripRider(ctx context.Context, tripID, riderID uuid.UUID) (bool, error)
	CompleteTrip(ctx context.Context, tripID uuid.UUID, fare float64, dropoffCode string) (bool, error)
	ListActiveTripsByRider(ctx context.Context, riderID uuid.UUID) ([]tripsModels.Trip, error)
}
