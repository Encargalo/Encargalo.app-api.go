package deliveryfee_test

import (
	"testing"
	"time"

	"Encargalo.app-api.go/internal/pkg/bizclock"
	"Encargalo.app-api.go/internal/shared/deliveryfee"
	"github.com/stretchr/testify/assert"
)

func atHour(hour int) time.Time {
	return time.Date(2026, time.October, 22, hour, 0, 0, 0, bizclock.Location())
}

// El delivery de comida cobra por kilómetro completo redondeado hacia arriba,
// con tasa plana. No usa el piso ni los tramos horarios de tripfare.
func TestFromDistanceAt_CobraPorKilometroRedondeadoHaciaArriba(t *testing.T) {
	cases := []struct {
		meters   float64
		expected float64
	}{
		{0, 0},
		{1, 0.40},
		{1000, 0.40},
		{1001, 0.80},
		{1500, 0.80},
		{2000, 0.80},
		{2001, 1.20},
		{4270, 2.00},
		{10804, 4.40},
	}

	for _, tc := range cases {
		assert.Equal(t, tc.expected, deliveryfee.FromDistanceAt(tc.meters, atHour(12)),
			"%.0f m diurno", tc.meters)
	}
}

func TestFromDistanceAt_TasaNocturnaDesdeLas22(t *testing.T) {
	assert.Equal(t, 0.40, deliveryfee.FromDistanceAt(1000, atHour(21)))
	assert.Equal(t, 0.60, deliveryfee.FromDistanceAt(1000, atHour(22)))
	assert.Equal(t, 0.60, deliveryfee.FromDistanceAt(1000, atHour(23)))
}

// Regresión: antes se leía time.Now().Hour() crudo y los contenedores corren en
// UTC, así que 22:00 UTC (18:00 en Caracas) disparaba la tasa nocturna.
func TestFromDistanceAt_UsaHoraDeNegocioNoUTC(t *testing.T) {
	utc22 := time.Date(2026, time.October, 22, 22, 0, 0, 0, time.UTC)
	local := utc22.In(bizclock.Location())

	assert.Equal(t, 18, local.Hour(), "22:00 UTC son 18:00 en zona de negocio")
	assert.Equal(t, 0.40, deliveryfee.FromDistanceAt(1000, local),
		"18:00 local es tasa diurna")
}
