package models

import (
	"time"

	"Encargalo.app-api.go/internal/pkg/money"
	"Encargalo.app-api.go/internal/shipments/domain/dtos"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type ShipmentStatus string

const (
	ShipmentStatusRequested     ShipmentStatus = "requested"
	ShipmentStatusPaymentReview ShipmentStatus = "payment_review"
	ShipmentStatusSearching     ShipmentStatus = "searching"
	ShipmentStatusAccepted      ShipmentStatus = "accepted"
	ShipmentStatusInTransit     ShipmentStatus = "in_transit"
	ShipmentStatusDelivered     ShipmentStatus = "delivered"
	ShipmentStatusCancelled     ShipmentStatus = "cancelled"
	ShipmentStatusNoDriver      ShipmentStatus = "no_driver"
)

// ReferenceTypeShipment identifica un envío dentro del módulo de pagos unificado.
const ReferenceTypeShipment = "shipment"

// CancelReason es el código que la app traduce a un texto para el cliente.
// Nunca lleva el motivo que escribió el staff ni dice quién canceló.
type CancelReason string

const (
	CancelReasonPaymentRejected   CancelReason = "payment_rejected"
	CancelReasonPaymentUnderpaid  CancelReason = "payment_underpaid"
	CancelReasonCustomerCancelled CancelReason = "customer_cancelled"
	CancelReasonNoDriver          CancelReason = "no_driver"
)

// Sin Efectivo: un envío solo se paga con Balance o Pago Móvil.
const MethodPagoMovil = "PagoMovil"
const MethodBalance = "Balance"

// CancellableStatuses son los estados desde los que un customer puede cancelar
// un envío. payment_review no está: el comprobante sigue en la cola del staff.
// in_transit/delivered/cancelled/no_driver tampoco.
var CancellableStatuses = map[ShipmentStatus]bool{
	ShipmentStatusRequested: true,
	ShipmentStatusSearching: true,
	ShipmentStatusAccepted:  true,
}

type Shipment struct {
	bun.BaseModel `bun:"table:shipments.shipments"`

	// ID siempre lo genera la app (UUIDv7 vía id.Ensure), nunca Postgres: a
	// diferencia de trips.trips, esta tabla no tiene DEFAULT uuid_generate_v4().
	ID uuid.UUID `bun:"id,pk,type:uuid"`
	// CustomerID es quien pidió el envío y paga, no necesariamente quien entrega o recibe.
	CustomerID uuid.UUID `bun:"customer_id,type:uuid,notnull"`
	// QuoteID enlaza el envío con la cotización que lo originó, para medir conversión.
	QuoteID          *uuid.UUID     `bun:"quote_id,type:uuid"`
	RiderID          *uuid.UUID     `bun:"rider_id,type:uuid"`
	Status           ShipmentStatus `bun:"status,notnull,default:'requested'"`
	PickupAddress    string         `bun:"pickup_address,notnull"`
	PickupLatitude   float64        `bun:"pickup_latitude,notnull"`
	PickupLongitude  float64        `bun:"pickup_longitude,notnull"`
	DropoffAddress   string         `bun:"dropoff_address,notnull"`
	DropoffLatitude  float64        `bun:"dropoff_latitude,notnull"`
	DropoffLongitude float64        `bun:"dropoff_longitude,notnull"`
	// *H3CellZone son resolución de analítica (heatmap), no de matching: el
	// matching de shipments es un broadcast a todo el pool en modo orders.
	PickupH3CellZone  *string `bun:"pickup_h3_cell_zone"`
	DropoffH3CellZone *string `bun:"dropoff_h3_cell_zone"`

	// Contactos: quién está físicamente presente en cada punto. IsCustomer en
	// true significa que es el propio CustomerID quien retira/recibe.
	SenderIsCustomer    bool   `bun:"sender_is_customer,notnull,default:true"`
	SenderName          string `bun:"sender_name,notnull"`
	SenderPhone         string `bun:"sender_phone,notnull"`
	RecipientIsCustomer bool   `bun:"recipient_is_customer,notnull,default:true"`
	RecipientName       string `bun:"recipient_name,notnull"`
	RecipientPhone      string `bun:"recipient_phone,notnull"`

	PackageDescription string `bun:"package_description,notnull"`

	Fare *float64 `bun:"fare"`
	// MethodPayment: PagoMovil o Balance. Sin Efectivo.
	MethodPayment string `bun:"method_payment,notnull,default:''"`
	// BcvRate es la tasa congelada en la cotización. ExpectedAmountBs = fare * bcv_rate.
	BcvRate       *float64 `bun:"bcv_rate"`
	BalancePaidBs *float64 `bun:"balance_paid_bs"`
	// Payment es el crédito a la cartera si el Pago Móvil aprobado no se prestó.
	Payment *ShipmentPayment `bun:"payment,type:jsonb"`
	// CancelReason es por qué terminó en cancelled o no_driver. Solo lo ve el cliente.
	CancelReason *CancelReason `bun:"cancel_reason"`

	// PickupCode/DropoffCode se generan ambos en el momento de creación (a
	// diferencia de trips, que los genera al aceptar/iniciar) y SIEMPRE se
	// exponen al cliente dueño del envío (ver ToCustomerDTO), sin importar el
	// estado: es él quien se los reenvía a quien esté físicamente presente.
	PickupCode        string     `bun:"pickup_code,notnull"`
	DropoffCode       string     `bun:"dropoff_code,notnull"`
	PickupCodeUsedAt  *time.Time `bun:"pickup_code_used_at"`
	DropoffCodeUsedAt *time.Time `bun:"dropoff_code_used_at"`

	// FareBs no se persiste, se recalcula en cada respuesta (patrón Trip.FareBs).
	FareBs      *float64   `bun:"-"`
	RequestedAt time.Time  `bun:"requested_at,notnull,default:current_timestamp"`
	AcceptedAt  *time.Time `bun:"accepted_at,nullzero"`
	DeliveredAt *time.Time `bun:"delivered_at,nullzero"`
	CancelledAt *time.Time `bun:"cancelled_at,nullzero"`
	CreatedAt   time.Time  `bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt   time.Time  `bun:"updated_at,notnull,default:current_timestamp"`
}

func (s *Shipment) IsCancellable() bool {
	return CancellableStatuses[s.Status]
}

// ApplyBsFare es no-op si todavía no hay Fare o la tasa no es válida.
func (s *Shipment) ApplyBsFare(rate float64) {
	if s.Fare == nil || rate <= 0 {
		return
	}
	bs := money.RoundBs(*s.Fare * rate)
	s.FareBs = &bs
}

// ApplyFrozenBsFare convierte con la tasa congelada en la cotización, la misma que vio el cliente.
func (s *Shipment) ApplyFrozenBsFare() {
	if s.BcvRate == nil {
		return
	}
	s.ApplyBsFare(*s.BcvRate)
}

// ToDTO alimenta el WebSocket del rider: nunca lleva pickup_code/dropoff_code,
// que el rider recibe de palabra de quien tenga el paquete en mano.
func (s *Shipment) ToDTO() dtos.ShipmentResponse {
	return dtos.ShipmentResponse{
		ID:                 s.ID,
		CustomerID:         s.CustomerID,
		RiderID:            s.RiderID,
		Status:             string(s.Status),
		PickupAddress:      s.PickupAddress,
		PickupLatitude:     s.PickupLatitude,
		PickupLongitude:    s.PickupLongitude,
		DropoffAddress:     s.DropoffAddress,
		DropoffLatitude:    s.DropoffLatitude,
		DropoffLongitude:   s.DropoffLongitude,
		Sender:             dtos.ShipmentContact{IsCustomer: s.SenderIsCustomer, Name: s.SenderName, Phone: s.SenderPhone},
		Recipient:          dtos.ShipmentContact{IsCustomer: s.RecipientIsCustomer, Name: s.RecipientName, Phone: s.RecipientPhone},
		PackageDescription: s.PackageDescription,
		FareUSD:            s.Fare,
		FareBs:             s.FareBs,
		MethodPayment:      s.MethodPayment,
		RequestedAt:        s.RequestedAt,
		AcceptedAt:         s.AcceptedAt,
		DeliveredAt:        s.DeliveredAt,
		CancelledAt:        s.CancelledAt,
	}
}

// ToCustomerDTO SIEMPRE incluye ambos códigos, sin condicionar por estado ni
// por UsedAt: decisión de producto, el cliente los reenvía manualmente a quien
// esté presente en retiro o entrega si no es él mismo.
func (s *Shipment) ToCustomerDTO() dtos.ShipmentResponse {
	resp := s.ToDTO()
	resp.PickupCode = s.PickupCode
	resp.DropoffCode = s.DropoffCode
	resp.Payment = s.Payment.toDTO()
	if s.CancelReason != nil {
		resp.CancelReason = string(*s.CancelReason)
	}
	return resp
}

// SetCancelReason no pisa un motivo ya escrito: el primero es la causa real.
func (s *Shipment) SetCancelReason(reason CancelReason) {
	if s == nil || s.CancelReason != nil {
		return
	}
	s.CancelReason = &reason
}
