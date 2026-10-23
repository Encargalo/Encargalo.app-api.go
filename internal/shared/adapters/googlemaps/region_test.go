package googlemaps

import "testing"

func TestMatchesRegion(t *testing.T) {
	cases := []struct {
		name        string
		countryCode string
		want        bool
	}{
		{"código del país que se atiende", "VE", true},
		{"mismo código en minúsculas", "ve", true},
		{"otro país", "CO", false},
		{"sin país en la respuesta", "", false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := matchesRegion(c.countryCode); got != c.want {
				t.Fatalf("matchesRegion(%q) = %v, se esperaba %v", c.countryCode, got, c.want)
			}
		})
	}
}
