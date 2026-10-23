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
	// NearbyRiders devuelve pines anónimos de conductores disponibles alrededor de un punto.
	NearbyRiders(lat, lng float64) (dtos.NearbyRidersResponse, error)
}

type TripsRepo interface {
	CreateTrip(ctx context.Context, trip *models.Trip) error
	GetTripByID(ctx context.Context, tripID uuid.UUID) (*models.Trip, error)
	GetTripHistoryByCustomer(ctx context.Context, customerID uuid.UUID) ([]models.Trip, error)
	// HasBillableTrip dice si el cliente tiene algún viaje fuera de cancelled/no_driver.
	HasBillableTrip(ctx context.Context, customerID uuid.UUID) (bool, error)
	UpdateTripStatus(ctx context.Context, tripID uuid.UUID, fromStatus, toStatus models.TripStatus) (bool, error)
	SaveTripPayment(ctx context.Context, tripID uuid.UUID, payment *models.TripPayment) error
	// SaveTripCancelReason escribe el motivo solo si el viaje aún no tiene uno.
	SaveTripCancelReason(ctx context.Context, tripID uuid.UUID, reason models.CancelReason) error
	AssignTripRider(ctx context.Context, tripID, riderID uuid.UUID, pickupCode string) (bool, error)
	// StartTripWithPickupCode pasa arriving→in_progress si el código de subida coincide
	// y guarda el código de bajada que el cliente mostrará al llegar.
	StartTripWithPickupCode(ctx context.Context, tripID, riderID uuid.UUID, pickupCode, dropoffCode string) (bool, error)
	ListActiveTripsByCustomer(ctx context.Context, customerID uuid.UUID) ([]models.Trip, error)
	// ListActiveTripsByRider devuelve los viajes asignados al rider que siguen
	// en curso (accepted, arriving o in_progress), el más reciente primero.
	ListActiveTripsByRider(ctx context.Context, riderID uuid.UUID) ([]models.Trip, error)
	// ReleaseTripRider devuelve el viaje a searching y limpia rider_id solo si
	// sigue asignado a ese rider en accepted o arriving. RowsAffected == 0 => el estado ya cambió.
	ReleaseTripRider(ctx context.Context, tripID, riderID uuid.UUID) (bool, error)
	// CompleteTrip hace un UPDATE condicional (WHERE status = in_progress) que
	// setea status = completed, fare y completed_at en un solo movimiento
	// atómico. RowsAffected == 0 => el estado ya cambió (CAS).
	CompleteTrip(ctx context.Context, tripID uuid.UUID, fare float64, dropoffCode string) (bool, error)
}

// CustomerConn es el WebSocket del cliente que sigue un viaje.
type CustomerConn interface {
	Send(msg []byte)
}

// CustomerTripNotifier empuja cambios de estado y la ubicación del rider al cliente conectado.
type CustomerTripNotifier interface {
	NotifyTrip(ctx context.Context, trip models.Trip)
	NotifyRiderLocation(ctx context.Context, customerID, tripID uuid.UUID, lat, lng float64)
}

// CustomerTripLive es el lado de conexión del mismo hub.
type CustomerTripLive interface {
	Register(ctx context.Context, customerID uuid.UUID, conn CustomerConn) error
	Unregister(customerID uuid.UUID, conn CustomerConn)
}

type TripRiderMatcher interface {
	OfferTrip(ctx context.Context, trip models.Trip)
}

// TripRiderNotifier avisa a los conductores cuando el cliente cancela.
// Lo implementa riders; trips no importa ese módulo.
// El mensaje no dice quién canceló ni el motivo.
type TripRiderNotifier interface {
	NotifyTripCancelled(ctx context.Context, trip models.Trip)
}

// RiderPublicInfoLookup resuelve los datos públicos del rider asignado.
// Lo implementa riders a partir de su perfil y de la cuenta, sin documentos KYC.
type RiderPublicInfoLookup interface {
	GetByRiderID(ctx context.Context, riderID uuid.UUID) (dtos.RiderPublicInfo, error)
}

// NearbyRidersLookup lee conductores en modo viajes cerca de un punto.
// Lo implementa el mismo proceso que guarda su ubicación en vivo.
type NearbyRidersLookup interface {
	NearbyRiders(lat, lng float64) ([]dtos.NearbyRider, error)
}

// TripQuotesRepo persiste TripQuote en Redis con TTL.
type TripQuotesRepo interface {
	Save(ctx context.Context, quote *models.TripQuote, ttl time.Duration) error
	// Get lee la cotización sin consumirla. Sirve para validar saldo antes de quemarla.
	Get(ctx context.Context, customerID, quoteID uuid.UUID) (*models.TripQuote, error)
	// GetAndConsume borra la cotización atómicamente al leerla (GETDEL); escopada por
	// customerID, así un quote_id ajeno nunca matchea (ni se borra) la de otro cliente.
	GetAndConsume(ctx context.Context, customerID, quoteID uuid.UUID) (*models.TripQuote, error)
}

// FareQuotesRepo persiste la cotización en OLTP para medir conversión. Es un
// registro de analítica: su escritura nunca debe hacer fallar la cotización.
type FareQuotesRepo interface {
	Insert(ctx context.Context, quote *models.FareQuote) error
}

// TripEventPublisher publica eventos de viaje al stream del servicio de notificaciones.
type TripEventPublisher interface {
	PublisherEvent(ctx context.Context, data any) error
}
