package dtos

import (
	"context"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

var shipmentValidate = validator.New()

// ShipmentContact es quién está físicamente presente en un extremo del envío.
// Si IsCustomer es true, Name/Phone se ignoran al crear: se toman de la cuenta
// del cliente. Si es false, Name/Phone son obligatorios: es un tercero.
type ShipmentContact struct {
	IsCustomer bool   `json:"is_customer"`
	Name       string `json:"name,omitempty"`
	Phone      string `json:"phone,omitempty"`
}

// CreateShipmentRequest exige la cotización vigente y el comprobante de Pago
// Móvil si aplica. Pickup, dropoff y fare salen de la cotización.
type CreateShipmentRequest struct {
	// ID es la llave de idempotencia: un reintento con el mismo id devuelve el envío ya creado.
	ID                uuid.UUID       `json:"id" validate:"omitempty,uuid" example:"3fa85f64-5717-4562-b3fc-2c963f66afa6"`
	QuoteID           uuid.UUID       `json:"quote_id" validate:"required,uuid"`
	MethodPayment     string          `json:"method_payment" validate:"required,oneof=PagoMovil Balance" example:"Balance"`
	PaymentProofImage string          `json:"payment_proof_image" validate:"max=7000000" example:"data:image/jpeg;base64,/9j/4AAQ..."`
	Sender            ShipmentContact `json:"sender" validate:"required"`
	Recipient         ShipmentContact `json:"recipient" validate:"required"`
	// PackageDescription es lo que se envía, en texto libre (ej. "una torta").
	PackageDescription string `json:"package_description" validate:"required,max=500" example:"Una torta de cumpleaños"`
	// PackagePhoto es opcional: una foto de lo que se envía, para que el rider
	// reconozca el paquete. A diferencia de payment_proof_image, nunca es obligatoria.
	PackagePhoto string `json:"package_photo,omitempty" validate:"omitempty,max=7000000" example:"data:image/jpeg;base64,/9j/4AAQ..."`
}

func (r *CreateShipmentRequest) Validate() error {
	if err := shipmentValidate.StructCtx(context.Background(), r); err != nil {
		return err
	}
	if r.MethodPayment == "PagoMovil" && len(r.PaymentProofImage) == 0 {
		return errRequired("payment_proof_image")
	}
	if !r.Sender.IsCustomer && (r.Sender.Name == "" || r.Sender.Phone == "") {
		return errRequired("sender.name y sender.phone")
	}
	if !r.Recipient.IsCustomer && (r.Recipient.Name == "" || r.Recipient.Phone == "") {
		return errRequired("recipient.name y recipient.phone")
	}
	return nil
}

func errRequired(field string) error {
	return &validationError{field: field}
}

type validationError struct{ field string }

func (e *validationError) Error() string { return e.field + " es requerido" }

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

type ShipmentResponse struct {
	ID                 uuid.UUID        `json:"id"`
	CustomerID         uuid.UUID        `json:"customer_id"`
	RiderID            *uuid.UUID       `json:"rider_id,omitempty"`
	Rider              *RiderPublicInfo `json:"rider,omitempty"`
	Status             string           `json:"status"`
	PickupAddress      string           `json:"pickup_address"`
	PickupLatitude     float64          `json:"pickup_latitude"`
	PickupLongitude    float64          `json:"pickup_longitude"`
	DropoffAddress     string           `json:"dropoff_address"`
	DropoffLatitude    float64          `json:"dropoff_latitude"`
	DropoffLongitude   float64          `json:"dropoff_longitude"`
	Sender             ShipmentContact  `json:"sender"`
	Recipient          ShipmentContact  `json:"recipient"`
	PackageDescription string           `json:"package_description"`
	PackagePhotoURL    *string          `json:"package_photo_url,omitempty"`
	FareUSD            *float64         `json:"fare_usd,omitempty"`
	FareBs             *float64         `json:"fare_bs,omitempty"`
	MethodPayment      string           `json:"method_payment,omitempty"`
	RequestedAt        time.Time        `json:"requested_at"`
	AcceptedAt         *time.Time       `json:"accepted_at,omitempty"`
	DeliveredAt        *time.Time       `json:"delivered_at,omitempty"`
	CancelledAt        *time.Time       `json:"cancelled_at,omitempty"`
	// PickupCode/DropoffCode siempre vienen presentes (lo genera el servidor al
	// crear el envío), sin importar el estado: el cliente los reenvía a quien
	// esté físicamente presente en retiro o entrega si no es él mismo.
	PickupCode  string           `json:"pickup_code,omitempty"`
	DropoffCode string           `json:"dropoff_code,omitempty"`
	Payment     *ShipmentPayment `json:"payment,omitempty"`
	// CancelReason es un código: payment_rejected, payment_underpaid,
	// customer_cancelled o no_driver. La app lo traduce; no dice quién canceló.
	CancelReason string `json:"cancel_reason,omitempty" example:"no_driver"`
}

// ShipmentPayment es lo que el cliente necesita saber de su dinero: el estado
// de la revisión del comprobante y el crédito a la cartera si el envío no se prestó.
type ShipmentPayment struct {
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

type ShipmentHistoryResponse struct {
	Shipments []ShipmentResponse `json:"shipments"`
}

// FareEstimateRequest congela todo lo necesario para crear el envío después.
// Sin VehicleType: la tarifa de envíos no depende del tipo de vehículo.
type FareEstimateRequest struct {
	CustomerID       uuid.UUID `json:"customer_id" swaggerignore:"true"`
	PickupAddress    string    `json:"pickup_address" validate:"required" example:"Av. Libertador, Caracas"`
	PickupLatitude   float64   `json:"pickup_latitude" validate:"required,latitude" example:"10.4806"`
	PickupLongitude  float64   `json:"pickup_longitude" validate:"required,longitude" example:"-66.9036"`
	DropoffAddress   string    `json:"dropoff_address" validate:"required" example:"CC Sambil, Caracas"`
	DropoffLatitude  float64   `json:"dropoff_latitude" validate:"required,latitude" example:"10.4956"`
	DropoffLongitude float64   `json:"dropoff_longitude" validate:"required,longitude" example:"-66.8517"`
}

func (r *FareEstimateRequest) Validate() error {
	return shipmentValidate.StructCtx(context.Background(), r)
}

type FareEstimateResponse struct {
	QuoteID        uuid.UUID `json:"quote_id"`
	DistanceMeters float64   `json:"distance_meters"`
	DurationMin    int64     `json:"duration_min"`
	FareUSD        float64   `json:"fare_usd"`
	FareBs         float64   `json:"fare_bs"`
	// BcvRate es la tasa congelada en la cotización; fare_bs = fare_usd * BcvRate.
	BcvRate   float64   `json:"bcv_rate"`
	ExpiresAt time.Time `json:"expires_at"`
}
