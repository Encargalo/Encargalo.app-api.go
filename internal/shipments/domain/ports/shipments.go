package ports

import (
	"context"
	"time"

	"Encargalo.app-api.go/internal/shipments/domain/dtos"
	"Encargalo.app-api.go/internal/shipments/domain/models"
	"github.com/google/uuid"
)

type ShipmentsApp interface {
	RequestShipment(ctx context.Context, customerID uuid.UUID, req dtos.CreateShipmentRequest) (dtos.ShipmentResponse, error)
	GetShipment(ctx context.Context, shipmentID, requesterID uuid.UUID) (dtos.ShipmentResponse, error)
	GetShipmentHistory(ctx context.Context, customerID uuid.UUID) (dtos.ShipmentHistoryResponse, error)
	CancelShipment(ctx context.Context, shipmentID, requesterID uuid.UUID) (dtos.ShipmentResponse, error)
	// EstimateFare congela el precio en una ShipmentQuote; RequestShipment la exige en vez de recalcular.
	EstimateFare(ctx context.Context, req dtos.FareEstimateRequest) (dtos.FareEstimateResponse, error)
}

type ShipmentsRepo interface {
	CreateShipment(ctx context.Context, shipment *models.Shipment) error
	GetShipmentByID(ctx context.Context, shipmentID uuid.UUID) (*models.Shipment, error)
	GetShipmentHistoryByCustomer(ctx context.Context, customerID uuid.UUID) ([]models.Shipment, error)
	UpdateShipmentStatus(ctx context.Context, shipmentID uuid.UUID, fromStatus, toStatus models.ShipmentStatus) (bool, error)
	SaveShipmentPayment(ctx context.Context, shipmentID uuid.UUID, payment *models.ShipmentPayment) error
	// SaveShipmentCancelReason escribe el motivo solo si el envío aún no tiene uno.
	SaveShipmentCancelReason(ctx context.Context, shipmentID uuid.UUID, reason models.CancelReason) error
	ListActiveShipmentsByCustomer(ctx context.Context, customerID uuid.UUID) ([]models.Shipment, error)

	// Los siguientes métodos, junto a los de arriba, completan el superconjunto
	// estructural que satisface ridersPorts.ShipmentsStore (riders/app los usa
	// a través de ese puerto más angosto, igual relación que TripsRepo/TripsStore).

	// AssignShipmentRider es un UPDATE condicional (WHERE status = searching)
	// que asigna el rider y pasa a accepted. RowsAffected == 0 => otro rider ganó.
	AssignShipmentRider(ctx context.Context, shipmentID, riderID uuid.UUID) (bool, error)
	// AdvanceShipmentStatus valida el código según la transición (pickup_code
	// para accepted->in_transit, dropoff_code para in_transit->delivered) y
	// hace el CAS correspondiente en un solo UPDATE. RowsAffected == 0 =>
	// el estado ya cambió o el código no coincide.
	AdvanceShipmentStatus(ctx context.Context, shipmentID, riderID uuid.UUID, code string) (bool, error)
	// ReleaseShipmentRider devuelve el envío a searching y limpia rider_id solo
	// si sigue asignado a ese rider en accepted. RowsAffected == 0 => el estado ya cambió.
	ReleaseShipmentRider(ctx context.Context, shipmentID, riderID uuid.UUID) (bool, error)
	// ListActiveShipmentsByRider devuelve los envíos asignados al rider que
	// siguen en curso (accepted o in_transit), el más reciente primero.
	ListActiveShipmentsByRider(ctx context.Context, riderID uuid.UUID) ([]models.Shipment, error)
	// ListSearchingShipmentIDs devuelve los envíos en "searching", para el
	// barrido de rescate que reprograma el vencimiento de los que quedaron sin uno.
	ListSearchingShipmentIDs(ctx context.Context) ([]uuid.UUID, error)
	// ListSearchingShipments trae los envíos en "searching" completos, para
	// reofertarlos a un rider que se reconecta (ver riders/app Register).
	ListSearchingShipments(ctx context.Context) ([]models.Shipment, error)
}

// CustomerConn es el WebSocket del cliente que sigue un envío.
type CustomerConn interface {
	Send(msg []byte)
}

// CustomerShipmentNotifier empuja cambios de estado al cliente conectado.
type CustomerShipmentNotifier interface {
	NotifyShipment(ctx context.Context, shipment models.Shipment)
	// NotifyRiderLocation empuja la ubicación en vivo del rider asignado, igual
	// que CustomerTripHub.NotifyRiderLocation para viajes.
	NotifyRiderLocation(ctx context.Context, customerID, shipmentID uuid.UUID, lat, lng float64)
}

// CustomerShipmentLive es el lado de conexión del mismo hub.
type CustomerShipmentLive interface {
	Register(ctx context.Context, customerID uuid.UUID, conn CustomerConn) error
	Unregister(customerID uuid.UUID, conn CustomerConn)
}

type ShipmentRiderMatcher interface {
	OfferShipment(ctx context.Context, shipment models.Shipment)
}

// ShipmentRiderNotifier avisa al rider asignado cuando el cliente cancela.
// Lo implementa riders; shipments no importa ese módulo.
type ShipmentRiderNotifier interface {
	NotifyShipmentCancelled(ctx context.Context, shipment models.Shipment)
}

// RiderPublicInfoLookup resuelve los datos públicos del rider asignado.
// Lo implementa riders a partir de su perfil y de la cuenta, sin documentos KYC.
type RiderPublicInfoLookup interface {
	GetByRiderID(ctx context.Context, riderID uuid.UUID) (dtos.RiderPublicInfo, error)
}

// CustomerContactLookup resuelve nombre y teléfono de la cuenta del cliente,
// para completar Sender/Recipient cuando IsCustomer es true. Mismo patrón que
// riders.PassengerLookup, pero implementado directamente en shipments/repo
// contra users.accounts (no hay necesidad de pasar por riders).
type CustomerContactLookup interface {
	GetByCustomerID(ctx context.Context, customerID uuid.UUID) (name, phone string, err error)
}

// ShipmentQuotesRepo persiste ShipmentQuote en Redis con TTL.
type ShipmentQuotesRepo interface {
	Save(ctx context.Context, quote *models.ShipmentQuote, ttl time.Duration) error
	// Get lee la cotización sin consumirla. Sirve para validar saldo antes de quemarla.
	Get(ctx context.Context, customerID, quoteID uuid.UUID) (*models.ShipmentQuote, error)
	// GetAndConsume borra la cotización atómicamente al leerla (GETDEL); escopada
	// por customerID, así un quote_id ajeno nunca matchea (ni se borra) la de otro cliente.
	GetAndConsume(ctx context.Context, customerID, quoteID uuid.UUID) (*models.ShipmentQuote, error)
}

// FareQuotesRepo persiste la cotización en OLTP para medir conversión. Es un
// registro de analítica: su escritura nunca debe hacer fallar la cotización.
type FareQuotesRepo interface {
	Insert(ctx context.Context, quote *models.FareQuote) error
}

// ShipmentEventPublisher publica eventos de envío al stream del servicio de notificaciones.
type ShipmentEventPublisher interface {
	PublisherEvent(ctx context.Context, data any) error
}
