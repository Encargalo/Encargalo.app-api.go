package ports

import (
	"context"
	"time"

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

// OrderCustomerNotifier entrega al cliente de una orden la ubicación en vivo de su rider.
// Lo implementa el manager WS de orders; se define aquí para no importar orders/app.
type OrderCustomerNotifier interface {
	NotifyRiderOrderLocation(ctx context.Context, customerID, orderID uuid.UUID, lat, lng float64, updatedAt time.Time)
}

// OrderCustomerNotifierSetter permite enlazar el notificador después de construir
// el manager (evita el ciclo de DI manager → OrdersApp → riders).
type OrderCustomerNotifierSetter interface {
	SetOrderCustomerNotifier(n OrderCustomerNotifier)
}

// PassengerLookup lee el nombre, el apellido y el teléfono de la cuenta del pasajero.
// No devuelve el email. El llamador decide si el teléfono sale al rider.
type PassengerLookup interface {
	GetByCustomerID(ctx context.Context, customerID uuid.UUID) (tripsDtos.PassengerContact, error)
}

// VehicleLookup lee el perfil del rider para conocer su vehículo. Subconjunto de RiderProfileRepo.
type VehicleLookup interface {
	GetByUserID(ctx context.Context, userID uuid.UUID) (*riderModels.RiderProfile, error)
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
	SaveTripPayment(ctx context.Context, tripID uuid.UUID, payment *tripsModels.TripPayment) error
	SaveTripCancelReason(ctx context.Context, tripID uuid.UUID, reason tripsModels.CancelReason) error
	// pickupDistanceKm congela la distancia del rider al pickup para liquidar el bono de acercamiento al completar.
	AssignTripRider(ctx context.Context, tripID, riderID uuid.UUID, pickupCode string, pickupDistanceKm float64) (bool, error)
	StartTripWithPickupCode(ctx context.Context, tripID, riderID uuid.UUID, pickupCode, dropoffCode string) (bool, error)
	ReleaseTripRider(ctx context.Context, tripID, riderID uuid.UUID) (bool, error)
	CompleteTrip(ctx context.Context, tripID uuid.UUID, fare float64, dropoffCode string) (bool, error)
	ListActiveTripsByRider(ctx context.Context, riderID uuid.UUID) ([]tripsModels.Trip, error)
	// ListSearchingTripIDs alimenta el barrido de rescate del
	// TripOfferTimeoutWorker (ver #359).
	ListSearchingTripIDs(ctx context.Context) ([]uuid.UUID, error)
}

// TripOfferTimeoutStore guarda fuera del proceso cuándo vence la ronda de
// oferta de cada viaje, para que un reinicio no la pierda (ver #359: antes
// vivía en un time.AfterFunc en memoria).
type TripOfferTimeoutStore interface {
	// Schedule programa (o reprograma, pisando el score anterior) el
	// vencimiento de tripID para el instante at.
	Schedule(ctx context.Context, tripID uuid.UUID, at time.Time) error
	// Cancel saca a tripID de la cola: se aceptó, se canceló, o ya no aplica.
	Cancel(ctx context.Context, tripID uuid.UUID) error
	// PopDue reclama hasta limit vencimientos ya cumplidos de forma atómica
	// (ZRANGEBYSCORE+ZREM en un solo script): con varias réplicas corriendo
	// esto al mismo tiempo, cada trip_id vencido solo puede salir una vez.
	PopDue(ctx context.Context, limit int64) ([]uuid.UUID, error)
	// ScheduleIfAbsent programa at solo si tripID no tiene ya un vencimiento
	// pendiente — lo usa el barrido de rescate para no adelantar el de un
	// viaje que ya está esperando su ronda real.
	ScheduleIfAbsent(ctx context.Context, tripID uuid.UUID, at time.Time) error
}
