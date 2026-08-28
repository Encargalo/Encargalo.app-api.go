package models

import (
	"time"

	"Encargalo.app-api.go/internal/trips/domain/dtos"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type TripStatus string

const (
	TripStatusRequested  TripStatus = "requested"
	TripStatusSearching  TripStatus = "searching"
	TripStatusAccepted   TripStatus = "accepted"
	TripStatusArriving   TripStatus = "arriving"
	TripStatusInProgress TripStatus = "in_progress"
	TripStatusCompleted  TripStatus = "completed"
	TripStatusCancelled  TripStatus = "cancelled"
	TripStatusNoDriver   TripStatus = "no_driver"
)

// CancellableStatuses son los estados desde los que un customer puede cancelar
// un trip; una vez in_progress/completed/cancelled/no_driver ya no aplica.
var CancellableStatuses = map[TripStatus]bool{
	TripStatusRequested: true,
	TripStatusSearching: true,
	TripStatusAccepted:  true,
	TripStatusArriving:  true,
}

type Trip struct {
	bun.BaseModel `bun:"table:trips.trips"`

	ID               uuid.UUID  `bun:"id,pk,type:uuid,default:uuid_generate_v4()"`
	CustomerID       uuid.UUID  `bun:"customer_id,type:uuid,notnull"`
	RiderID          *uuid.UUID `bun:"rider_id,type:uuid"`
	Status           TripStatus `bun:"status,notnull,default:'requested'"`
	PickupAddress    string     `bun:"pickup_address,notnull"`
	PickupLatitude   float64    `bun:"pickup_latitude,notnull"`
	PickupLongitude  float64    `bun:"pickup_longitude,notnull"`
	PickupH3Cell     *string    `bun:"pickup_h3_cell"`
	DropoffAddress   string     `bun:"dropoff_address,notnull"`
	DropoffLatitude  float64    `bun:"dropoff_latitude,notnull"`
	DropoffLongitude float64    `bun:"dropoff_longitude,notnull"`
	Fare             *float64   `bun:"fare"`
	RequestedAt      time.Time  `bun:"requested_at,notnull,default:current_timestamp"`
	AcceptedAt       *time.Time `bun:"accepted_at,nullzero"`
	CompletedAt      *time.Time `bun:"completed_at,nullzero"`
	CancelledAt      *time.Time `bun:"cancelled_at,nullzero"`
	CreatedAt        time.Time  `bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt        time.Time  `bun:"updated_at,notnull,default:current_timestamp"`
}

func (t *Trip) IsCancellable() bool {
	return CancellableStatuses[t.Status]
}

func (t *Trip) ToDTO() dtos.TripResponse {
	return dtos.TripResponse{
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
		Fare:             t.Fare,
		RequestedAt:      t.RequestedAt,
		AcceptedAt:       t.AcceptedAt,
		CompletedAt:      t.CompletedAt,
		CancelledAt:      t.CancelledAt,
	}
}
