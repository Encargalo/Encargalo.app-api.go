package models

import riderModels "Encargalo.app-api.go/internal/riders/domain/models"

// TripService es un vehículo que el cliente puede elegir para su viaje.
// El tipo es el mismo valor que riders.profiles.vehicle_type: el matching los compara.
type TripService struct {
	Type     string `json:"type"`
	Name     string `json:"name"`
	Capacity int    `json:"capacity"`
	// Available en false deja el servicio visible pero no pedible.
	Available bool `json:"available"`
}

// TripServices es el catálogo. El carro nace en el contrato pero no disponible:
// sin riders con ese vehículo, un viaje en carro expiraría sin conductor.
var TripServices = []TripService{
	{Type: riderModels.VehicleTypeMoto, Name: "Moto", Capacity: 1, Available: true},
	{Type: riderModels.VehicleTypeCar, Name: "Carro", Capacity: 4, Available: false},
}

// DefaultTripService es lo que se cotiza cuando el cliente no manda el tipo.
// La app vieja no lo manda y debe seguir funcionando.
const DefaultTripService = riderModels.VehicleTypeMoto

// TripServiceAvailable indica si se puede cotizar un viaje con ese vehículo.
func TripServiceAvailable(vehicleType string) bool {
	for _, s := range TripServices {
		if s.Type == vehicleType {
			return s.Available
		}
	}
	return false
}
