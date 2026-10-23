package deliveryfee

import (
	"context"
	"errors"
	"time"

	"Encargalo.app-api.go/internal/pkg/bizclock"
	"Encargalo.app-api.go/internal/pkg/money"
	"Encargalo.app-api.go/internal/shared/adapters/googlemaps"
)

const (
	// MinFare es el piso de toda tarifa: cubre cualquier trayecto de hasta IncludedKm.
	MinFare = 1.00

	// IncludedKm son los kilómetros que MinFare ya cubre sin cargo extra.
	// Calibrado contra precios de calle: un trayecto de 4,27 km cuesta $1.
	IncludedKm = 4.5

	// ExtraRatePerKm se cobra por cada kilómetro por encima de IncludedKm.
	// Calibrado contra precios de calle: 10,8 km cuesta $5.
	ExtraRatePerKm = 0.63

	// Multiplicadores por tramo horario. La madrugada es el tramo más caro
	// porque ahí el limitante es la oferta: casi no hay riders despiertos y el
	// regreso vacío es casi seguro.
	NightMultiplier        = 1.5
	EarlyMorningMultiplier = 1.7

	// Bordes de los tramos: nocturno 22:00-01:59, madrugada 02:00-03:59,
	// diurno 04:00-21:59.
	NightStartHour        = 22
	EarlyMorningStartHour = 2
	DayStartHour          = 4

	// DevPlaceholderFee es el delivery fee fijo que se muestra/cobra fuera de
	// production, para no gastar cuota de la API de Google Maps en desarrollo.
	// Debe usarse consistentemente en todo lugar donde se calcule o se le
	// muestre al cliente un delivery fee (listados de tiendas, creación de
	// órdenes, etc.), para que el monto que ve el cliente antes de pagar
	// coincida con el que el backend cobra al confirmar.
	DevPlaceholderFee = 2.0
)

// Band es el tramo horario que determina el multiplicador de la tarifa.
type Band string

const (
	BandDay          Band = "day"
	BandNight        Band = "night"
	BandEarlyMorning Band = "early_morning"
)

// BandAt resuelve el tramo horario. `now` debe venir en la zona horaria de
// negocio (bizclock.Now()), no en UTC.
// El nocturno cruza medianoche, por eso su rango se evalúa como dos extremos y
// no como un intervalo.
func BandAt(now time.Time) Band {
	hour := now.Hour()
	if hour >= NightStartHour || hour < EarlyMorningStartHour {
		return BandNight
	}
	if hour < DayStartHour {
		return BandEarlyMorning
	}
	return BandDay
}

// Multiplier es el factor que el tramo aplica sobre la tarifa.
func (b Band) Multiplier() float64 {
	switch b {
	case BandNight:
		return NightMultiplier
	case BandEarlyMorning:
		return EarlyMorningMultiplier
	default:
		return 1
	}
}

// FromDistance convierte una distancia en metros a tarifa.
// El cobro es proporcional a la distancia exacta: se redondea el dinero, no los
// kilómetros, para que 4.001 m no cueste lo mismo que 4.999 m.
func FromDistance(distanceMeters float64) float64 {
	return FromDistanceAt(distanceMeters, bizclock.Now())
}

// FromDistanceAt es FromDistance con el reloj inyectado. `now` debe venir en la
// zona horaria de negocio (bizclock.Now()), no en UTC.
func FromDistanceAt(distanceMeters float64, now time.Time) float64 {
	km := distanceMeters / 1000

	fare := MinFare
	if km > IncludedKm {
		fare += (km - IncludedKm) * ExtraRatePerKm
	}

	return money.RoundUSD(fare * BandAt(now).Multiplier())
}

// Calculate obtiene la distancia vial entre tienda y dirección de entrega y retorna la tarifa.
func Calculate(ctx context.Context, shopLat, shopLon, deliveryLat, deliveryLon float64) (float64, error) {
	distMeters, err := googlemaps.DistanceMeters(ctx, shopLat, shopLon, deliveryLat, deliveryLon)
	if err != nil {
		return 0, err
	}
	return FromDistance(distMeters), nil
}

// Estimate agrega la distancia y duración estimadas del trayecto junto con la
// tarifa calculada y el tramo horario que la produjo.
type Estimate struct {
	DistanceMeters float64
	DurationMin    int64
	Fee            float64
	Band           Band
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
	// Un solo Now para tarifa y tramo: leerlo dos veces podría registrar un
	// tramo distinto del multiplicador que se aplicó.
	now := bizclock.Now()
	elem := results[0]
	return Estimate{
		DistanceMeters: elem.DistanceMeters,
		DurationMin:    elem.DurationMin,
		Fee:            FromDistanceAt(elem.DistanceMeters, now),
		Band:           BandAt(now),
	}, nil
}
