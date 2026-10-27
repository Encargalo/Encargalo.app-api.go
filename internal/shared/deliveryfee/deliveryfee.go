package deliveryfee

import (
	"context"
	"time"

	"Encargalo.app-api.go/internal/pkg/bizclock"
	"Encargalo.app-api.go/internal/shared/adapters/googlemaps"
	"Encargalo.app-api.go/internal/shared/tripfare"
)

// El delivery de comida usa la misma fórmula que los viajes de pasajeros
// (internal/shared/tripfare): piso, kilómetros incluidos y tramos horarios.
const (
	// DevPlaceholderFee es el delivery fee fijo que se muestra/cobra fuera de
	// production, para no gastar cuota de la API de Google Maps en desarrollo.
	// Debe usarse consistentemente en todo lugar donde se calcule o se le
	// muestre al cliente un delivery fee (listados de tiendas, creación de
	// órdenes, etc.), para que el monto que ve el cliente antes de pagar
	// coincida con el que el backend cobra al confirmar.
	DevPlaceholderFee = tripfare.DevPlaceholderFee
)

// Estimate agrega la distancia y el tramo horario junto con la tarifa
// calculada, para que quien la invoque pueda registrar la cotización durable
// sin recalcular contra Google Maps.
type Estimate struct {
	DistanceMeters float64
	Fee            float64
	Band           tripfare.Band
}

// FromDistance convierte una distancia en metros a tarifa de entrega.
func FromDistance(distanceMeters float64) float64 {
	return FromDistanceAt(distanceMeters, bizclock.Now())
}

// FromDistanceAt es FromDistance con el reloj inyectado. `now` debe venir en la
// zona horaria de negocio (bizclock.Now()), no en UTC.
func FromDistanceAt(distanceMeters float64, now time.Time) float64 {
	return tripfare.FromDistanceAt(distanceMeters, now)
}

// Calculate obtiene la distancia vial entre tienda y dirección de entrega y retorna la tarifa.
func Calculate(ctx context.Context, shopLat, shopLon, deliveryLat, deliveryLon float64) (float64, error) {
	distMeters, err := googlemaps.DistanceMeters(ctx, shopLat, shopLon, deliveryLat, deliveryLon)
	if err != nil {
		return 0, err
	}
	return FromDistance(distMeters), nil
}

// CalculateWithBand es Calculate pero además devuelve la distancia y el tramo
// horario que produjeron la tarifa, para poder registrar la cotización durable.
func CalculateWithBand(ctx context.Context, shopLat, shopLon, deliveryLat, deliveryLon float64) (Estimate, error) {
	distMeters, err := googlemaps.DistanceMeters(ctx, shopLat, shopLon, deliveryLat, deliveryLon)
	if err != nil {
		return Estimate{}, err
	}
	now := bizclock.Now()
	return Estimate{
		DistanceMeters: distMeters,
		Fee:            FromDistanceAt(distMeters, now),
		Band:           tripfare.BandAt(now),
	}, nil
}
