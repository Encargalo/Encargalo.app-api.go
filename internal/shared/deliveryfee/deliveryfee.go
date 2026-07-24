package deliveryfee

import (
	"math"
	"time"

	"Encargalo.app-api.go/internal/shared/adapters/googlemaps"
)

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
	km := math.Ceil(distanceMeters / 1000)
	rate := float64(RateDay)
	if time.Now().Hour() >= NightHour {
		rate = RateNight
	}
	return km * rate
}

// Calculate obtiene la distancia vial entre tienda y dirección de entrega y retorna la tarifa.
func Calculate(shopLat, shopLon, deliveryLat, deliveryLon float64) (float64, error) {
	distMeters, err := googlemaps.DistanceMeters(shopLat, shopLon, deliveryLat, deliveryLon)
	if err != nil {
		return 0, err
	}
	return FromDistance(distMeters), nil
}
