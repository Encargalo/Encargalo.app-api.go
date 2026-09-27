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

const (
	// NearbyRiderRings es el radio de búsqueda en anillos H3 de resolución 8 (~2 km).
	NearbyRiderRings = 2
	// NearbyRiderMaxResults acota la respuesta: el mapa no necesita la flota entera.
	NearbyRiderMaxResults = 30
	// NearbyRiderMaxAge descarta ubicaciones que dejaron de reportarse.
	NearbyRiderMaxAge = 45 * time.Second
	// NearbyCoordDecimals redondea a 0.001° (~111 m) para no publicar el GPS crudo.
	NearbyCoordDecimals = 1000
)
