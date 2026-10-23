package deliveryfee_test

import (
	"testing"
	"time"

	"Encargalo.app-api.go/internal/pkg/bizclock"
	"Encargalo.app-api.go/internal/shared/deliveryfee"
	"github.com/stretchr/testify/assert"
)

// mediodía en zona de negocio, fuera del tramo nocturno.
func atHour(hour int) time.Time {
	return time.Date(2026, time.October, 22, hour, 0, 0, 0, bizclock.Now().Location())
}

// Anclas tomadas de precios reales de calle en Charallave/Santa Teresa,
// con la distancia vial que devuelve Google Distance Matrix.
func TestFromDistanceAt_AnclasDeMercado(t *testing.T) {
	cases := []struct {
		name     string
		meters   float64
		expected float64
	}{
		{"Santa Marta -> Pueblo Nuevo (calle $1.00)", 4270, 1.00},
		{"Pueblo Nuevo -> Cada Sta Rosa (calle $1.00)", 4438, 1.00},
		{"Santa Marta -> Terminal Charallave (calle $5.00)", 10804, 4.97},
		{"Terminal Charallave -> Santa Marta (calle $5.00)", 9580, 4.20},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, deliveryfee.FromDistanceAt(tc.meters, atHour(12)))
		})
	}
}

func TestFromDistanceAt_PisoCubreHastaIncludedKm(t *testing.T) {
	cases := []float64{0, 1, 500, 1000, 2000, 4000, 4500}

	for _, meters := range cases {
		assert.Equal(t, 1.00, deliveryfee.FromDistanceAt(meters, atHour(12)),
			"todo trayecto de hasta 4,5 km cobra el piso")
	}
}

// El cobro es proporcional: cruzar un kilómetro entero no debe crear un escalón.
func TestFromDistanceAt_SinEscalonesPorKilometro(t *testing.T) {
	cases := []struct {
		meters   float64
		expected float64
	}{
		{4501, 1.00},
		{4671, 1.11},
		{5000, 1.32},
		{5001, 1.32},
		{6000, 1.95},
		{8000, 3.21},
	}

	for _, tc := range cases {
		assert.Equal(t, tc.expected, deliveryfee.FromDistanceAt(tc.meters, atHour(12)))
	}
}

// Tramos: nocturno 22:00-01:59 (x1,5), madrugada 02:00-03:59 (x1,7),
// diurno 04:00-21:59 (x1,0).
func TestFromDistanceAt_TramosHorarios(t *testing.T) {
	cases := []struct {
		hour     int
		expected float64
		tramo    string
	}{
		{0, 1.50, "nocturno cruzando medianoche"},
		{1, 1.50, "nocturno"},
		{2, 1.70, "madrugada"},
		{3, 1.70, "madrugada"},
		{4, 1.00, "diurno"},
		{12, 1.00, "diurno"},
		{21, 1.00, "diurno"},
		{22, 1.50, "nocturno"},
		{23, 1.50, "nocturno"},
	}

	for _, tc := range cases {
		assert.Equal(t, tc.expected, deliveryfee.FromDistanceAt(4270, atHour(tc.hour)),
			"%02d:00 debe ser %s", tc.hour, tc.tramo)
	}
}

// El multiplicador escala la tarifa completa, distancia incluida.
func TestFromDistanceAt_TramosSobreViajeLargo(t *testing.T) {
	assert.Equal(t, 4.97, deliveryfee.FromDistanceAt(10804, atHour(12)))
	assert.Equal(t, 7.46, deliveryfee.FromDistanceAt(10804, atHour(22)))
	assert.Equal(t, 8.45, deliveryfee.FromDistanceAt(10804, atHour(2)))
}

// Regresión: antes se leía time.Now().Hour() crudo y los contenedores corren en
// UTC, así que 22:00 UTC (18:00 en Caracas) disparaba la tarifa nocturna.
func TestFromDistanceAt_UsaHoraDeNegocioNoUTC(t *testing.T) {
	utc22 := time.Date(2026, time.October, 22, 22, 0, 0, 0, time.UTC)
	local := utc22.In(bizclock.Now().Location())

	assert.Equal(t, 18, local.Hour(), "22:00 UTC son 18:00 en zona de negocio")
	assert.Equal(t, 1.00, deliveryfee.FromDistanceAt(4270, local),
		"18:00 local es tarifa diurna")
}
