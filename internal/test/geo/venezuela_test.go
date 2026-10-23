package geo_test

import (
	"testing"

	"Encargalo.app-api.go/internal/pkg/geo"
)

func TestInVenezuela(t *testing.T) {
	cases := []struct {
		name string
		lat  float64
		lng  float64
		want bool
	}{
		{"Caracas", 10.48801, -66.87919, true},
		{"Maracaibo", 10.65427, -71.64125, true},
		{"Puerto Ayacucho", 5.66375, -67.62687, true},
		{"Isla de Margarita", 10.99897, -63.86842, true},
		{"Santa Elena de Uairén", 4.60294, -61.11024, true},
		{"Bogotá", 4.60971, -74.08175, false},
		{"Manaos", -3.11903, -60.02173, false},
		{"Miami", 25.76168, -80.19179, false},
		{"Georgetown", 6.80137, -58.15527, false},
		{"origen del plano", 0, 0, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := geo.InVenezuela(c.lat, c.lng); got != c.want {
				t.Fatalf("InVenezuela(%v, %v) = %v, se esperaba %v", c.lat, c.lng, got, c.want)
			}
		})
	}
}
