package models

import (
	"time"

	"github.com/google/uuid"
)

type RiderMode string

const (
	RiderModeOrders RiderMode = "orders"
	RiderModeTrips  RiderMode = "trips"
)

type RiderLocation struct {
	Latitude  float64
	Longitude float64
	H3Cell    string
	UpdatedAt time.Time
}

type RiderLocationEntry struct {
	RiderID  uuid.UUID
	Location RiderLocation
}
