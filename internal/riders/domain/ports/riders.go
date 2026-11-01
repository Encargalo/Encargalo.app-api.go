package ports

import (
	"context"
	"time"

	riderModels "Encargalo.app-api.go/internal/riders/domain/models"
	shipmentsModels "Encargalo.app-api.go/internal/shipments/domain/models"
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
	// ReportLocation es el equivalente de UpdateLocation para cuando no hay
	// socket vivo (la app en background) — ver #push-notifications-background.
	ReportLocation(ctx context.Context, riderID uuid.UUID, lat, lng float64, mode riderModels.RiderMode) error
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
	// SetAvailability pausa o reanuda la recepción de nuevas órdenes y viajes.
	// El rider sigue conectado y conserva lo que ya tiene asignado.
	SetAvailability(ctx context.Context, riderID uuid.UUID, acceptingNewWork bool) error
	// AcceptShipment asigna el envío por CAS en BD (sin candado en memoria: a
	// diferencia de trips, el pool "orders" permite varias asignaciones
	// simultáneas por rider). Si otro rider ya ganó, manda shipment_taken.
	AcceptShipment(ctx context.Context, riderID, shipmentID uuid.UUID, conn RiderConn) error
	// AdvanceShipment avanza accepted→in_transit (valida pickup_code) o
	// in_transit→delivered (valida dropoff_code, acredita la ganancia del rider).
	AdvanceShipment(ctx context.Context, riderID, shipmentID uuid.UUID, code string, conn RiderConn) error
	// CancelAcceptedShipment libera un envío accepted y lo vuelve a ofertar a
	// todo el pool en modo orders (sin lista de exclusión: el broadcast ya es abierto).
	CancelAcceptedShipment(ctx context.Context, riderID, shipmentID uuid.UUID, conn RiderConn) error
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

// ShipmentsStore is the slice of shipments persistence the riders domain needs
// to offer and accept shipments — narrower than the full
// shipments/domain/ports.ShipmentsRepo surface, which also covers shipment
// creation and history unrelated to matching. Igual relación que TripsStore.
type ShipmentsStore interface {
	GetShipmentByID(ctx context.Context, shipmentID uuid.UUID) (*shipmentsModels.Shipment, error)
	UpdateShipmentStatus(ctx context.Context, shipmentID uuid.UUID, fromStatus, toStatus shipmentsModels.ShipmentStatus) (bool, error)
	SaveShipmentPayment(ctx context.Context, shipmentID uuid.UUID, payment *shipmentsModels.ShipmentPayment) error
	SaveShipmentCancelReason(ctx context.Context, shipmentID uuid.UUID, reason shipmentsModels.CancelReason) error
	AssignShipmentRider(ctx context.Context, shipmentID, riderID uuid.UUID) (bool, error)
	AdvanceShipmentStatus(ctx context.Context, shipmentID, riderID uuid.UUID, code string) (bool, error)
	ReleaseShipmentRider(ctx context.Context, shipmentID, riderID uuid.UUID) (bool, error)
	ListActiveShipmentsByRider(ctx context.Context, riderID uuid.UUID) ([]shipmentsModels.Shipment, error)
	// ListSearchingShipmentIDs alimenta el barrido de rescate del
	// ShipmentOfferTimeoutWorker, análogo a ListSearchingTripIDs.
	ListSearchingShipmentIDs(ctx context.Context) ([]uuid.UUID, error)
	// ListSearchingShipments trae los envíos en searching completos, para
	// reofertarlos a un rider que se reconecta (ver Register).
	ListSearchingShipments(ctx context.Context) ([]shipmentsModels.Shipment, error)
}

// ShipmentOfferTimeoutStore guarda fuera del proceso cuándo vence la espera de
// oferta de cada envío, para que un reinicio no la pierda. Análogo
// simplificado de TripOfferTimeoutStore: un único vencimiento por envío, sin
// rondas, porque el broadcast ya salió a todo el pool desde la primera oferta.
type ShipmentOfferTimeoutStore interface {
	Schedule(ctx context.Context, shipmentID uuid.UUID, at time.Time) error
	Cancel(ctx context.Context, shipmentID uuid.UUID) error
	PopDue(ctx context.Context, limit int64) ([]uuid.UUID, error)
	ScheduleIfAbsent(ctx context.Context, shipmentID uuid.UUID, at time.Time) error
}
