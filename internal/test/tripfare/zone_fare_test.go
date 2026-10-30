package tripfare_test

import (
	"testing"

	"Encargalo.app-api.go/internal/pkg/money"
	"Encargalo.app-api.go/internal/shared/tripfare"
	"github.com/stretchr/testify/assert"
)

// charallaveZone son los parámetros reales acordados con negocio (issue
// #376): IncludedKm único para las tres bandas, mínima y tarifa/km propias
// por banda, sin el multiplicador global.
var charallaveZone = &tripfare.ZoneFareParams{
	IncludedKm:                 3,
	DayMinFare:                 1.2375,
	DayExtraRatePerKm:          0.30,
	NightMinFare:               1.575,
	NightExtraRatePerKm:        0.50,
	EarlyMorningMinFare:        1.80,
	EarlyMorningExtraRatePerKm: 0.70,
}

func TestFromDistanceAtZone_Charallave(t *testing.T) {
	cases := []struct {
		name     string
		km       float64
		hour     int
		expected float64
	}{
		{"día, dentro de included_km", 3.0, 12, 1.24},
		{"día, 3.5 km", 3.5, 12, 1.39},
		{"día, 4.5 km", 4.5, 12, 1.69},
		{"día, 6 km", 6.0, 12, 2.14},
		{"día, 10 km", 10.0, 12, 3.34},
		{"noche, dentro de included_km", 3.0, 23, 1.58},
		{"noche, 3.5 km", 3.5, 23, 1.83},
		{"noche, 4.5 km", 4.5, 23, 2.33},
		{"noche, 6 km", 6.0, 23, 3.08},
		{"noche, 10 km", 10.0, 23, 5.08},
		{"madrugada, dentro de included_km", 3.0, 3, 1.80},
		{"madrugada, 3.5 km", 3.5, 3, 2.15},
		{"madrugada, 4.5 km", 4.5, 3, 2.85},
		{"madrugada, 6 km", 6.0, 3, 3.90},
		{"madrugada, 10 km", 10.0, 3, 6.70},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tripfare.FromDistanceAtZone(tc.km*1000, atHour(tc.hour), charallaveZone)
			assert.Equal(t, tc.expected, got)
		})
	}
}

// El multiplicador global de banda (Night 1.5x, EarlyMorning 1.7x) no se
// aplica dentro de una zona: cada banda ya trae su mínima y tarifa finales.
func TestFromDistanceAtZone_DoesNotApplyGlobalBandMultiplier(t *testing.T) {
	night := tripfare.FromDistanceAtZone(3000, atHour(23), charallaveZone)
	assert.Equal(t, 1.58, night) // money.RoundUSD(1.575), sin el 1.5x global encima
	assert.NotEqual(t, money.RoundUSD(charallaveZone.NightMinFare*tripfare.NightMultiplier), night)
}

// Sin zona (nil), el comportamiento es exactamente el de FromDistanceAt: el
// fallback global sigue intacto para quien no pasa por una zona (deliveryfee).
func TestFromDistanceAtZone_NilFallsBackToGlobalFormula(t *testing.T) {
	now := atHour(12)
	got := tripfare.FromDistanceAtZone(10000, now, nil)
	want := tripfare.FromDistanceAt(10000, now)
	assert.Equal(t, want, got)
}
