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

// El delivery de comida usa la misma fórmula que tripfare: piso de $1 que
// cubre hasta 4.5 km, y $0.63 extra por cada km por encima de ese piso.
func TestFromDistanceAt_UsaFormulaDeTripfare(t *testing.T) {
	cases := []struct {
		meters   float64
		expected float64
	}{
		{0, 1.00},
		{1, 1.00},
		{4500, 1.00},
		{4501, 1.00},
		{5000, 1.32},
		{10804, 4.97},
	}

	for _, tc := range cases {
		assert.Equal(t, tc.expected, deliveryfee.FromDistanceAt(tc.meters, atHour(12)),
			"%.0f m diurno", tc.meters)
	}
}

func TestFromDistanceAt_TasaNocturnaDesdeLas22(t *testing.T) {
	assert.Equal(t, 1.00, deliveryfee.FromDistanceAt(1000, atHour(21)))
	assert.Equal(t, 1.50, deliveryfee.FromDistanceAt(1000, atHour(22)))
	assert.Equal(t, 1.50, deliveryfee.FromDistanceAt(1000, atHour(23)))
}

// Regresión: antes se leía time.Now().Hour() crudo y los contenedores corren en
// UTC, así que 22:00 UTC (18:00 en Caracas) disparaba la tasa nocturna.
func TestFromDistanceAt_UsaHoraDeNegocioNoUTC(t *testing.T) {
	utc22 := time.Date(2026, time.October, 22, 22, 0, 0, 0, time.UTC)
	local := utc22.In(bizclock.Location())

	assert.Equal(t, 18, local.Hour(), "22:00 UTC son 18:00 en zona de negocio")
	assert.Equal(t, 1.00, deliveryfee.FromDistanceAt(1000, local),
		"18:00 local es tasa diurna")
}
