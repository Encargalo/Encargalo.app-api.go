package dtos

import (
	"context"
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

var tripValidate = validator.New()

// CreateTripRequest exige la cotización vigente y el comprobante de Pago Móvil.
// Pickup, dropoff y fare salen de la cotización; no se recalculan.
type CreateTripRequest struct {
	// ID es la llave de idempotencia: un reintento con el mismo id devuelve el viaje ya creado.
	ID                uuid.UUID `json:"id" validate:"omitempty,uuid" example:"3fa85f64-5717-4562-b3fc-2c963f66afa6"`
	QuoteID           uuid.UUID `json:"quote_id" validate:"required,uuid"`
	MethodPayment     string    `json:"method_payment" validate:"required,oneof=PagoMovil Balance Efectivo" example:"PagoMovil"`
	PaymentProofImage string    `json:"payment_proof_image" validate:"max=7000000" example:"data:image/jpeg;base64,/9j/4AAQ..."`
	// BalanceAmountBs y CashAmountBs parten la tarifa. Cero en ambos deja el
	// método único de siempre. Con un monto, el otro método es el resto.
	BalanceAmountBs float64 `json:"balance_amount_bs" example:"80"`
	CashAmountBs    float64 `json:"cash_amount_bs" example:"120"`
}

func (r *CreateTripRequest) Validate() error {
	if err := tripValidate.StructCtx(context.Background(), r); err != nil {
		return err
	}
	if r.MethodPayment == "PagoMovil" && len(r.PaymentProofImage) == 0 && r.CashAmountBs <= 0 && r.BalanceAmountBs <= 0 {
		return fmt.Errorf("payment_proof_image es requerido")
	}
	if r.BalanceAmountBs < 0 || r.CashAmountBs < 0 {
		return fmt.Errorf("los montos no pueden ser negativos")
	}
	return nil
}

// PassengerContact es el nombre, el apellido y el teléfono del pasajero.
// El nombre y el apellido llegan al rider desde la oferta; el teléfono solo
// al rider que aceptó el viaje, mientras sigue en curso (accepted, arriving
// o in_progress).
type PassengerContact struct {
	Name     string `json:"name"`
	LastName string `json:"last_name,omitempty"`
	Phone    string `json:"phone,omitempty"`
}

// RiderPublicInfo es lo que el cliente puede ver del rider asignado.
// No incluye documentos KYC (cédula, carnet, licencia, carnet de circulación).
type RiderPublicInfo struct {
	Name            string  `json:"name"`
	SurName         string  `json:"sur_name,omitempty"`
	Phone           string  `json:"phone,omitempty"`
	ProfilePhotoURL *string `json:"profile_photo_url,omitempty"`
	VehicleType     *string `json:"vehicle_type,omitempty"`
	VehiclePlate    *string `json:"vehicle_plate,omitempty"`
	VehicleBrand    *string `json:"vehicle_brand,omitempty"`
	VehicleModel    *string `json:"vehicle_model,omitempty"`
	VehicleColor    *string `json:"vehicle_color,omitempty"`
}

type TripResponse struct {
	ID               uuid.UUID         `json:"id"`
	CustomerID       uuid.UUID         `json:"customer_id"`
	RiderID          *uuid.UUID        `json:"rider_id,omitempty"`
	Rider            *RiderPublicInfo  `json:"rider,omitempty"`
	Customer         *PassengerContact `json:"customer,omitempty"`
	Status           string            `json:"status"`
	PickupAddress    string            `json:"pickup_address"`
	PickupLatitude   float64           `json:"pickup_latitude"`
	PickupLongitude  float64           `json:"pickup_longitude"`
	DropoffAddress   string            `json:"dropoff_address"`
	DropoffLatitude  float64           `json:"dropoff_latitude"`
	DropoffLongitude float64           `json:"dropoff_longitude"`
	FareUSD          *float64          `json:"fare_usd,omitempty"`
	FareBs           *float64          `json:"fare_bs,omitempty"`
	DiscountUSD      float64           `json:"discount_usd,omitempty"`
	MethodPayment    string            `json:"method_payment,omitempty"`
	// CollectUSD es lo que el rider cobra en efectivo al pasajero. Su pago ya
	// va dentro de ese monto; el resto queda como deuda en su balance.
	CollectUSD *float64 `json:"collect_usd,omitempty"`
	// CollectBs es CollectUSD en bolívares con la tasa congelada del viaje.
	CollectBs   *float64     `json:"collect_bs,omitempty"`
	RequestedAt time.Time    `json:"requested_at"`
	AcceptedAt  *time.Time   `json:"accepted_at,omitempty"`
	StartedAt   *time.Time   `json:"started_at,omitempty"`
	CompletedAt *time.Time   `json:"completed_at,omitempty"`
	CancelledAt *time.Time   `json:"cancelled_at,omitempty"`
	PickupCode  *string      `json:"pickup_code,omitempty"`
	DropoffCode *string      `json:"dropoff_code,omitempty"`
	Payment     *TripPayment `json:"payment,omitempty"`
	// CancelReason es un código: payment_rejected, payment_underpaid,
	// customer_cancelled o no_driver. La app lo traduce; no dice quién canceló.
	CancelReason string `json:"cancel_reason,omitempty" example:"no_driver"`
}

// TripPayment es lo que el cliente necesita saber de su dinero: el estado de la
// revisión del comprobante y el crédito a la cartera si el viaje no se prestó.
type TripPayment struct {
	// Status es el estado de la revisión: approved, overpaid,
	// rejected_underpaid o rejected_invalid. Vacío si no hubo revisión.
	Status           string   `json:"status,omitempty" example:"rejected_underpaid"`
	ExpectedAmountBs float64  `json:"expected_amount_bs,omitempty"`
	ObservedAmountBs *float64 `json:"observed_amount_bs,omitempty"`
	DifferenceBs     *float64 `json:"difference_bs,omitempty"`
	// RejectionReason solo se llena en rejected_underpaid. El motivo que
	// escribe el staff en rejected_invalid nunca sale hacia el cliente.
	RejectionReason     *string    `json:"rejection_reason,omitempty"`
	TransactionID       *uuid.UUID `json:"transaction_id,omitempty"`
	CreditedBs          *float64   `json:"credited_bs,omitempty"`
	RefundablePMBs      *float64   `json:"refundable_pm_bs,omitempty"`
	CanRequestRefund    bool       `json:"can_request_refund,omitempty"`
	RefundEligibleUntil *string    `json:"refund_eligible_until,omitempty"`
}

type TripHistoryResponse struct {
	Trips []TripResponse `json:"trips"`
}

// NearbyRider es un conductor disponible en el mapa de cotización.
// La coordenada está redondeada (~100 m) y no incluye id ni datos del perfil.
type NearbyRider struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type NearbyRidersResponse struct {
	Riders []NearbyRider `json:"riders"`
}

// TripServiceResponse es un servicio del catálogo. available en false lo deja
// visible pero no pedible: la app lo pinta deshabilitado.
type TripServiceResponse struct {
	Type      string `json:"type" example:"moto"`
	Name      string `json:"name" example:"Moto"`
	Capacity  int    `json:"capacity" example:"1"`
	Available bool   `json:"available" example:"true"`
}

type TripServicesResponse struct {
	Services []TripServiceResponse `json:"services"`
}

// FareEstimateRequest congela todo lo necesario para crear el viaje después.
type FareEstimateRequest struct {
	CustomerID       uuid.UUID `json:"customer_id" swaggerignore:"true"`
	PickupAddress    string    `json:"pickup_address" validate:"required" example:"Av. Libertador, Caracas"`
	PickupLatitude   float64   `json:"pickup_latitude" validate:"required,latitude" example:"10.4806"`
	PickupLongitude  float64   `json:"pickup_longitude" validate:"required,longitude" example:"-66.9036"`
	DropoffAddress   string    `json:"dropoff_address" validate:"required" example:"CC Sambil, Caracas"`
	DropoffLatitude  float64   `json:"dropoff_latitude" validate:"required,latitude" example:"10.4956"`
	DropoffLongitude float64   `json:"dropoff_longitude" validate:"required,longitude" example:"-66.8517"`
	// VehicleType es opcional: sin él se cotiza moto, que es lo que manda la app actual.
	VehicleType string `json:"vehicle_type" validate:"omitempty,oneof=moto carro" example:"moto"`
}

func (r *FareEstimateRequest) Validate() error {
	return tripValidate.StructCtx(context.Background(), r)
}

type FareEstimateResponse struct {
	QuoteID        uuid.UUID `json:"quote_id"`
	DistanceMeters float64   `json:"distance_meters"`
	DurationMin    int64     `json:"duration_min"`
	FareUSD        float64   `json:"fare_usd"`
	FareBs         float64   `json:"fare_bs"`
	// FareOriginalUSD y DiscountUSD solo vienen con el descuento de primer viaje; fare_usd ya lo trae restado.
	FareOriginalUSD float64 `json:"fare_original_usd,omitempty"`
	DiscountUSD     float64 `json:"discount_usd,omitempty"`
	// BcvRate es la tasa congelada en la cotización; fare_bs = fare_usd * BcvRate.
	BcvRate float64 `json:"bcv_rate"`
	// VehicleType es el servicio que quedó congelado en la cotización.
	VehicleType string    `json:"vehicle_type"`
	ExpiresAt   time.Time `json:"expires_at"`
}
