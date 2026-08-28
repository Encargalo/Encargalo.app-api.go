package dtos

import (
	"context"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

var tripValidate = validator.New()

type CreateTripRequest struct {
	PickupAddress    string  `json:"pickup_address" validate:"required" example:"Av. Libertador, Caracas"`
	PickupLatitude   float64 `json:"pickup_latitude" validate:"required,latitude" example:"10.4806"`
	PickupLongitude  float64 `json:"pickup_longitude" validate:"required,longitude" example:"-66.9036"`
	DropoffAddress   string  `json:"dropoff_address" validate:"required" example:"CC Sambil, Caracas"`
	DropoffLatitude  float64 `json:"dropoff_latitude" validate:"required,latitude" example:"10.4956"`
	DropoffLongitude float64 `json:"dropoff_longitude" validate:"required,longitude" example:"-66.8517"`
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
	Fare             *float64   `json:"fare,omitempty"`
	RequestedAt      time.Time  `json:"requested_at"`
	AcceptedAt       *time.Time `json:"accepted_at,omitempty"`
	CompletedAt      *time.Time `json:"completed_at,omitempty"`
	CancelledAt      *time.Time `json:"cancelled_at,omitempty"`
}

type TripHistoryResponse struct {
	Trips []TripResponse `json:"trips"`
}
