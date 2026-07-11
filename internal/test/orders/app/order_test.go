package app_test

import (
	"testing"

	"Encargalo.app-api.go/internal/orders/app"
	"github.com/stretchr/testify/assert"
)

// PriceMismatch protege el checkout de cobrar un total distinto al que el
// front mostró al cliente (ver ErrPriceChanged en orderApp.CreateOrder). Se
// testea aisladamente porque CreateOrder en sí depende de deliveryfee.Calculate,
// que hace una llamada real a Google Maps y no es mockeable sin refactorizarla
// a un port inyectable.
func TestPriceMismatch(t *testing.T) {
	cases := []struct {
		name     string
		actual   float64
		expected float64
		want     bool
	}{
		{"exact match", 22.0, 22.0, false},
		{"within tolerance", 22.0, 22.005, false},
		{"just over tolerance", 22.0, 22.02, true},
		{"launch promo applied but front expected full price", 12.0, 22.0, true},
		{"stale discount: front expected less than server total", 22.0, 12.0, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, app.PriceMismatch(tc.actual, tc.expected))
		})
	}
}
