// Package tripfare calcula la tarifa de los viajes de pasajeros.
// internal/shared/deliveryfee (delivery de comida) reusa esta misma fórmula.
package tripfare

import (
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

	// DevPlaceholderFee es la tarifa fija que se usa fuera de production, para
	// no gastar cuota de la API de Google Maps en desarrollo.
	DevPlaceholderFee = 2.0

	// PickupDistanceThresholdKm es el acercamiento al pickup a partir del cual ya no lo absorbe el rider solo.
	PickupDistanceThresholdKm = 3.0

	// PickupBonusRatePerKm es lo que se le paga al rider por km de acercamiento una vez superado el umbral.
	PickupBonusRatePerKm = 0.40

	// PickupUserSurchargeKm son los km de acercamiento que el cargo al usuario cubre, como monto fijo.
	PickupUserSurchargeKm = 2.0
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

// FromDistance convierte una distancia en metros a tarifa de viaje.
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

// PickupBonus paga al rider su distancia real de acercamiento; 0 si no supera el umbral.
func PickupBonus(pickupDistanceKm float64) float64 {
	if pickupDistanceKm <= PickupDistanceThresholdKm {
		return 0
	}
	return money.RoundUSD(pickupDistanceKm * PickupBonusRatePerKm)
}

// PickupSurcharge es un monto fijo, no escala con la distancia real, para no trasladarle al usuario la variabilidad de qué rider termine aceptando.
func PickupSurcharge(avgPickupDistanceKm float64) float64 {
	if avgPickupDistanceKm <= PickupDistanceThresholdKm {
		return 0
	}
	return money.RoundUSD(PickupUserSurchargeKm * PickupBonusRatePerKm)
}

// Estimate agrega la distancia y duración estimadas del trayecto junto con la
// tarifa calculada y el tramo horario que la produjo.
type Estimate struct {
	DistanceMeters float64
	DurationMin    int64
	Fee            float64
	Band           Band
}

// CalculateWithDuration obtiene la distancia vial y la duración del trayecto
// junto con la tarifa calculada.
func CalculateWithDuration(originLat, originLon, destLat, destLon float64) (Estimate, error) {
	return CalculateWithDurationZone(originLat, originLon, destLat, destLon, nil)
}

// ZoneFareParams son los montos de una zona geográfica (ej. Charallave),
// resueltos por quien llama (trips/app, consultando ZoneFareRulesRepo por la
// celda H3 del pickup) antes de pasarlos acá: este paquete no conoce esa
// tabla ni hace I/O. A diferencia del fallback global, estos montos por
// banda ya son finales — no se multiplican por Band.Multiplier().
type ZoneFareParams struct {
	IncludedKm                 float64
	DayMinFare                 float64
	DayExtraRatePerKm          float64
	NightMinFare               float64
	NightExtraRatePerKm        float64
	EarlyMorningMinFare        float64
	EarlyMorningExtraRatePerKm float64
}

func (z ZoneFareParams) forBand(b Band) (minFare, extraRatePerKm float64) {
	switch b {
	case BandNight:
		return z.NightMinFare, z.NightExtraRatePerKm
	case BandEarlyMorning:
		return z.EarlyMorningMinFare, z.EarlyMorningExtraRatePerKm
	default:
		return z.DayMinFare, z.DayExtraRatePerKm
	}
}

// FromDistanceAtZone es FromDistanceAt con la zona resuelta inyectada. zone
// nil cae en el fallback global (igual que FromDistanceAt, multiplicador de
// banda incluido); zone no nil usa la mínima/tarifa ya finales de esa banda.
func FromDistanceAtZone(distanceMeters float64, now time.Time, zone *ZoneFareParams) float64 {
	if zone == nil {
		return FromDistanceAt(distanceMeters, now)
	}

	km := distanceMeters / 1000
	minFare, extraRate := zone.forBand(BandAt(now))
	fare := minFare
	if km > zone.IncludedKm {
		fare += (km - zone.IncludedKm) * extraRate
	}
	return money.RoundUSD(fare)
}

// CalculateWithDurationZone es CalculateWithDuration con la zona resuelta
// inyectada. Ver FromDistanceAtZone.
func CalculateWithDurationZone(originLat, originLon, destLat, destLon float64, zone *ZoneFareParams) (Estimate, error) {
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
		Fee:            FromDistanceAtZone(elem.DistanceMeters, now, zone),
		Band:           BandAt(now),
	}, nil
}
