package shipmentfare

import (
	"time"

	"Encargalo.app-api.go/internal/pkg/bizclock"
	"Encargalo.app-api.go/internal/shared/tripfare"
)

// Los envíos usan la misma fórmula que los viajes de pasajeros
// (internal/shared/tripfare): piso, kilómetros incluidos y tramos horarios.
// Mismo precedente que internal/shared/deliveryfee para delivery de comida.
const DevPlaceholderFee = tripfare.DevPlaceholderFee

// FromDistance convierte una distancia en metros a tarifa de envío.
func FromDistance(distanceMeters float64) float64 {
	return FromDistanceAt(distanceMeters, bizclock.Now())
}

// FromDistanceAt es FromDistance con el reloj inyectado. `now` debe venir en la
// zona horaria de negocio (bizclock.Now()), no en UTC.
func FromDistanceAt(distanceMeters float64, now time.Time) float64 {
	return tripfare.FromDistanceAt(distanceMeters, now)
}

// CalculateWithDuration obtiene la distancia/duración vial y la tarifa entre
// origen y destino, sin reglas de zona (v1 usa siempre el fallback global).
func CalculateWithDuration(pickupLat, pickupLon, dropoffLat, dropoffLon float64) (tripfare.Estimate, error) {
	return tripfare.CalculateWithDurationZone(pickupLat, pickupLon, dropoffLat, dropoffLon, nil)
}
