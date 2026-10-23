package deliveryfee

import (
	"context"
	"math"
	"time"

	"Encargalo.app-api.go/internal/pkg/bizclock"
	"Encargalo.app-api.go/internal/pkg/money"
	"Encargalo.app-api.go/internal/shared/adapters/googlemaps"
)

// Esta tarifa es solo para el delivery de comida. Los viajes de pasajeros usan
// internal/shared/tripfare, que tiene piso, kilómetros incluidos y tramos
// horarios: son dos negocios con economía distinta y no comparten política.
const (
	RateDay   = 0.40
	RateNight = 0.60
	NightHour = 22

	// DevPlaceholderFee es el delivery fee fijo que se muestra/cobra fuera de
	// production, para no gastar cuota de la API de Google Maps en desarrollo.
	// Debe usarse consistentemente en todo lugar donde se calcule o se le
	// muestre al cliente un delivery fee (listados de tiendas, creación de
	// órdenes, etc.), para que el monto que ve el cliente antes de pagar
	// coincida con el que el backend cobra al confirmar.
	DevPlaceholderFee = 2.0
)

// FromDistance convierte una distancia en metros a tarifa de entrega.
// El cobro es por km completo redondeado hacia arriba (ej: 1001m = 2 km = $0.80 diurno).
func FromDistance(distanceMeters float64) float64 {
	return FromDistanceAt(distanceMeters, bizclock.Now())
}

// FromDistanceAt es FromDistance con el reloj inyectado. `now` debe venir en la
// zona horaria de negocio (bizclock.Now()), no en UTC.
func FromDistanceAt(distanceMeters float64, now time.Time) float64 {
	km := math.Ceil(distanceMeters / 1000)
	rate := float64(RateDay)
	if now.Hour() >= NightHour {
		rate = RateNight
	}
	// Redondeo para no serializarle 1.2000000000000002 al cliente; no cambia
	// el precio, solo su representación.
	return money.RoundUSD(km * rate)
}

// Calculate obtiene la distancia vial entre tienda y dirección de entrega y retorna la tarifa.
func Calculate(ctx context.Context, shopLat, shopLon, deliveryLat, deliveryLon float64) (float64, error) {
	distMeters, err := googlemaps.DistanceMeters(ctx, shopLat, shopLon, deliveryLat, deliveryLon)
	if err != nil {
		return 0, err
	}
	return FromDistance(distMeters), nil
}
