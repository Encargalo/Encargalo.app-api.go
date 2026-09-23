package dtos

import tripsDtos "Encargalo.app-api.go/internal/trips/domain/dtos"

// ActiveTripsResponse son los viajes que el rider ya tiene asignados y siguen en curso.
type ActiveTripsResponse struct {
	Trips []tripsDtos.TripResponse `json:"trips"`
}
