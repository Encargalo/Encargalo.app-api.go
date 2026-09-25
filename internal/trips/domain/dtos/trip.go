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
	QuoteID           uuid.UUID `json:"quote_id" validate:"required,uuid4"`
	MethodPayment     string    `json:"method_payment" validate:"required,oneof=PagoMovil Balance" example:"PagoMovil"`
	PaymentProofImage string    `json:"payment_proof_image" validate:"max=7000000" example:"data:image/jpeg;base64,/9j/4AAQ..."`
}

func (r *CreateTripRequest) Validate() error {
	if err := tripValidate.StructCtx(context.Background(), r); err != nil {
		return err
	}
	if r.MethodPayment == "PagoMovil" && len(r.PaymentProofImage) == 0 {
		return fmt.Errorf("payment_proof_image es requerido")
	}
	return nil
}

// PassengerContact es el nombre y el teléfono del pasajero.
// Solo lo recibe el rider que aceptó el viaje, mientras sigue en curso
// (accepted, arriving o in_progress). No incluye apellido.
type PassengerContact struct {
	Name  string `json:"name"`
	Phone string `json:"phone,omitempty"`
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
	RequestedAt      time.Time         `json:"requested_at"`
	AcceptedAt       *time.Time        `json:"accepted_at,omitempty"`
	CompletedAt      *time.Time        `json:"completed_at,omitempty"`
	CancelledAt      *time.Time        `json:"cancelled_at,omitempty"`
	PickupCode       *string           `json:"pickup_code,omitempty"`
	DropoffCode      *string           `json:"dropoff_code,omitempty"`
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
