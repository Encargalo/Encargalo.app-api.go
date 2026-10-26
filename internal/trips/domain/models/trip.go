package models

import (
	"math"
	"time"

	"Encargalo.app-api.go/internal/pkg/money"
	"Encargalo.app-api.go/internal/trips/domain/dtos"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type TripStatus string

const (
	TripStatusRequested     TripStatus = "requested"
	TripStatusPaymentReview TripStatus = "payment_review"
	TripStatusSearching     TripStatus = "searching"
	TripStatusAccepted      TripStatus = "accepted"
	TripStatusArriving      TripStatus = "arriving"
	TripStatusInProgress    TripStatus = "in_progress"
	TripStatusCompleted     TripStatus = "completed"
	TripStatusCancelled     TripStatus = "cancelled"
	TripStatusNoDriver      TripStatus = "no_driver"
)

// ReferenceTypeTrip identifica un viaje dentro del módulo de pagos unificado.
const ReferenceTypeTrip = "trip"

// CancelReason es el código que la app traduce a un texto para el cliente.
// Nunca lleva el motivo que escribió el staff ni dice quién canceló.
type CancelReason string

const (
	CancelReasonPaymentRejected   CancelReason = "payment_rejected"
	CancelReasonPaymentUnderpaid  CancelReason = "payment_underpaid"
	CancelReasonCustomerCancelled CancelReason = "customer_cancelled"
	CancelReasonNoDriver          CancelReason = "no_driver"
)

const MethodPagoMovil = "PagoMovil"
const MethodBalance = "Balance"
const MethodEfectivo = "Efectivo"

// El primer viaje lleva 50% de descuento con tope de $6, absorbido por la plataforma.
const (
	FirstTripDiscountRate   = 0.5
	FirstTripDiscountCapUSD = 6.0
)

// FirstTripDiscountUSD es el descuento sobre la tarifa completa del primer viaje.
func FirstTripDiscountUSD(grossUSD float64) float64 {
	if grossUSD <= 0 {
		return 0
	}
	return money.RoundUSD(math.Min(grossUSD*FirstTripDiscountRate, FirstTripDiscountCapUSD))
}

// CancellableStatuses son los estados desde los que un customer puede cancelar
// un trip. payment_review no está: el comprobante sigue en la cola del staff.
// in_progress/completed/cancelled/no_driver tampoco.
var CancellableStatuses = map[TripStatus]bool{
	TripStatusRequested: true,
	TripStatusSearching: true,
	TripStatusAccepted:  true,
	TripStatusArriving:  true,
}

type Trip struct {
	bun.BaseModel `bun:"table:trips.trips"`

	ID         uuid.UUID `bun:"id,pk,type:uuid,default:uuid_generate_v4()"`
	CustomerID uuid.UUID `bun:"customer_id,type:uuid,notnull"`
	// QuoteID enlaza el viaje con la cotización que lo originó; es el que permite
	// medir qué cotizaciones convirtieron. Nulo en viajes previos a trips.fare_quotes.
	QuoteID          *uuid.UUID `bun:"quote_id,type:uuid"`
	RiderID          *uuid.UUID `bun:"rider_id,type:uuid"`
	Status           TripStatus `bun:"status,notnull,default:'requested'"`
	PickupAddress    string     `bun:"pickup_address,notnull"`
	PickupLatitude   float64    `bun:"pickup_latitude,notnull"`
	PickupLongitude  float64    `bun:"pickup_longitude,notnull"`
	PickupH3Cell     *string    `bun:"pickup_h3_cell"`
	DropoffAddress   string     `bun:"dropoff_address,notnull"`
	DropoffLatitude  float64    `bun:"dropoff_latitude,notnull"`
	DropoffLongitude float64    `bun:"dropoff_longitude,notnull"`
	// A diferencia de PickupH3Cell (DriverMatchResolution), usan ZoneAnalyticsResolution para el heatmap.
	PickupH3CellZone  *string  `bun:"pickup_h3_cell_zone"`
	DropoffH3CellZone *string  `bun:"dropoff_h3_cell_zone"`
	Fare              *float64 `bun:"fare"`
	// DiscountUSD es el descuento de primer viaje. Fare ya lo trae restado; el rider cobra sobre Fare + DiscountUSD.
	DiscountUSD float64 `bun:"discount_usd,notnull,default:0"`

	// VehicleType es el servicio con el que se cotizó; el matching filtra por él.
	VehicleType string `bun:"vehicle_type,notnull"`
	// MethodPayment distingue PagoMovil, Balance y Efectivo. Vacío en viajes
	// creados antes de persistir el método: esos se liquidan como bancarios.
	MethodPayment string `bun:"method_payment,notnull,default:''"`
	// BcvRate es la tasa congelada en la cotización. ExpectedAmountBs = fare * bcv_rate.
	BcvRate *float64 `bun:"bcv_rate"`
	// BalancePaidBs es el débito al saldo. CashAmountBs es la parte que cobra el rider.
	BalancePaidBs *float64 `bun:"balance_paid_bs"`
	CashAmountBs  *float64 `bun:"cash_amount_bs"`
	// Payment es el crédito a la cartera si el Pago Móvil aprobado no se prestó.
	Payment *TripPayment `bun:"payment,type:jsonb"`
	// CancelReason es por qué terminó en cancelled o no_driver. Solo lo ve el cliente.
	CancelReason      *CancelReason `bun:"cancel_reason"`
	PickupCode        *string       `bun:"pickup_code"`
	DropoffCode       *string       `bun:"dropoff_code"`
	PickupCodeUsedAt  *time.Time    `bun:"pickup_code_used_at"`
	DropoffCodeUsedAt *time.Time    `bun:"dropoff_code_used_at"`
	// FareBs no se persiste, se recalcula en cada respuesta (patrón Order.DeliveryFeeBs).
	FareBs      *float64   `bun:"-"`
	RequestedAt time.Time  `bun:"requested_at,notnull,default:current_timestamp"`
	AcceptedAt  *time.Time `bun:"accepted_at,nullzero"`
	StartedAt   *time.Time `bun:"started_at,nullzero"`
	CompletedAt *time.Time `bun:"completed_at,nullzero"`
	CancelledAt *time.Time `bun:"cancelled_at,nullzero"`
	CreatedAt   time.Time  `bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt   time.Time  `bun:"updated_at,notnull,default:current_timestamp"`
}

func (t *Trip) IsCancellable() bool {
	return CancellableStatuses[t.Status]
}

// GrossFareUSD es la tarifa completa antes del descuento, la base del pago del rider.
func (t *Trip) GrossFareUSD() float64 {
	if t.Fare == nil {
		return 0
	}
	return money.RoundUSD(*t.Fare + t.DiscountUSD)
}

// ApplyBsFare es no-op si todavía no hay Fare o la tasa no es válida.
func (t *Trip) ApplyBsFare(rate float64) {
	if t.Fare == nil || rate <= 0 {
		return
	}
	bs := money.RoundBs(*t.Fare * rate)
	t.FareBs = &bs
}

// ApplyFrozenBsFare convierte con la tasa congelada en la cotización, la misma que vio el pasajero.
func (t *Trip) ApplyFrozenBsFare() {
	if t.BcvRate == nil {
		return
	}
	t.ApplyBsFare(*t.BcvRate)
}

func (t *Trip) ToDTO() dtos.TripResponse {
	resp := dtos.TripResponse{
		ID:               t.ID,
		CustomerID:       t.CustomerID,
		RiderID:          t.RiderID,
		Status:           string(t.Status),
		PickupAddress:    t.PickupAddress,
		PickupLatitude:   t.PickupLatitude,
		PickupLongitude:  t.PickupLongitude,
		DropoffAddress:   t.DropoffAddress,
		DropoffLatitude:  t.DropoffLatitude,
		DropoffLongitude: t.DropoffLongitude,
		FareUSD:          t.Fare,
		FareBs:           t.FareBs,
		DiscountUSD:      t.DiscountUSD,
		MethodPayment:    t.MethodPayment,
		RequestedAt:      t.RequestedAt,
		AcceptedAt:       t.AcceptedAt,
		StartedAt:        t.StartedAt,
		CompletedAt:      t.CompletedAt,
		CancelledAt:      t.CancelledAt,
	}
	if t.CashAmountBs != nil && *t.CashAmountBs > 0 && t.BcvRate != nil && *t.BcvRate > 0 {
		usd := money.RoundUSD(*t.CashAmountBs / *t.BcvRate)
		resp.CollectUSD = &usd
	}
	if resp.CollectUSD == nil && t.MethodPayment == MethodEfectivo && t.Fare != nil {
		fare := *t.Fare
		resp.CollectUSD = &fare
	}
	if t.CashAmountBs != nil && *t.CashAmountBs > 0 {
		bs := *t.CashAmountBs
		resp.CollectBs = &bs
	}
	if resp.CollectBs == nil && t.MethodPayment == MethodEfectivo && t.Fare != nil && t.BcvRate != nil && *t.BcvRate > 0 {
		bs := money.RoundBs(*t.Fare * *t.BcvRate)
		resp.CollectBs = &bs
	}
	return resp
}

// ToCustomerDTO incluye el código que el cliente debe mostrar. El rider no lo
// recibe: ToDTO, que alimenta su WebSocket, no copia estos campos.
func (t *Trip) ToCustomerDTO() dtos.TripResponse {
	resp := t.ToDTO()
	if t.PickupCodeUsedAt == nil {
		resp.PickupCode = t.PickupCode
	}
	if t.Status == TripStatusInProgress && t.DropoffCodeUsedAt == nil {
		resp.DropoffCode = t.DropoffCode
	}
	resp.Payment = t.Payment.toDTO()
	if t.CancelReason != nil {
		resp.CancelReason = string(*t.CancelReason)
	}
	return resp
}

// SetCancelReason no pisa un motivo ya escrito: el primero es la causa real.
func (t *Trip) SetCancelReason(reason CancelReason) {
	if t == nil || t.CancelReason != nil {
		return
	}
	t.CancelReason = &reason
}
