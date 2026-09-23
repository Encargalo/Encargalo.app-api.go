package dtos

import (
	"context"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

var tripValidate = validator.New()

// CreateTripRequest exige la cotización vigente y el comprobante de Pago Móvil.
// Pickup, dropoff y fare salen de la cotización; no se recalculan.
type CreateTripRequest struct {
	QuoteID           uuid.UUID `json:"quote_id" validate:"required,uuid4"`
	MethodPayment     string    `json:"method_payment" validate:"required,oneof=PagoMovil" example:"PagoMovil"`
	PaymentProofImage string    `json:"payment_proof_image" validate:"required,max=7000000" example:"data:image/jpeg;base64,/9j/4AAQ..."`
}

func (r *CreateTripRequest) Validate() error {
	return tripValidate.StructCtx(context.Background(), r)
}

type TripResponse struct {
	ID               uuid.UUID  `json:"id"`
	CustomerID       uuid.UUID  `json:"customer_id"`
	RiderID          *uuid.UUID `json:"rider_id,omitempty"`
	Status           string     `json:"status"`
	PickupAddress    string     `json:"pickup_address"`
	PickupLatitude   float64    `json:"pickup_latitude"`
	PickupLongitude  float64    `json:"pickup_longitude"`
	DropoffAddress   string     `json:"dropoff_address"`
	DropoffLatitude  float64    `json:"dropoff_latitude"`
	DropoffLongitude float64    `json:"dropoff_longitude"`
	FareUSD          *float64   `json:"fare_usd,omitempty"`
	FareBs           *float64   `json:"fare_bs,omitempty"`
	RequestedAt      time.Time  `json:"requested_at"`
	AcceptedAt       *time.Time `json:"accepted_at,omitempty"`
	CompletedAt      *time.Time `json:"completed_at,omitempty"`
	CancelledAt      *time.Time `json:"cancelled_at,omitempty"`
}

type TripHistoryResponse struct {
	Trips []TripResponse `json:"trips"`
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
	ExpiresAt      time.Time `json:"expires_at"`
}
