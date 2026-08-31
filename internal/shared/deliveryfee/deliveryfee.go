package deliveryfee

import (
	"context"
	"errors"
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
func Calculate(ctx context.Context, shopLat, shopLon, deliveryLat, deliveryLon float64) (float64, error) {
	distMeters, err := googlemaps.DistanceMeters(ctx, shopLat, shopLon, deliveryLat, deliveryLon)
	if err != nil {
		return 0, err
	}
	return FromDistance(distMeters), nil
}

// Estimate agrega la distancia y duración estimadas del trayecto junto con la tarifa calculada.
type Estimate struct {
	DistanceMeters float64
	DurationMin    int64
	Fee            float64
}

// CalculateWithDuration es como Calculate, pero además retorna la duración estimada.
func CalculateWithDuration(originLat, originLon, destLat, destLon float64) (Estimate, error) {
	results, err := googlemaps.DistanceMatrix(originLat, originLon, [][2]float64{{destLat, destLon}})
	if err != nil {
		return Estimate{}, err
	}
	if len(results) == 0 {
		return Estimate{}, errors.New("google maps: no se pudo resolver el destino")
	}
	elem := results[0]
	return Estimate{
		DistanceMeters: elem.DistanceMeters,
		DurationMin:    elem.DurationMin,
		Fee:            FromDistance(elem.DistanceMeters),
	}, nil
}
