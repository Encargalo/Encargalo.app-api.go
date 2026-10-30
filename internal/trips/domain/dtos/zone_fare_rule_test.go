package dtos

import "testing"

func validZoneFareRuleRequest() ZoneFareRuleRequest {
	return ZoneFareRuleRequest{
		Name:                   "Charallave",
		IncludedKm:             3,
		DayMinFareUSD:          1.2375,
		NightMinFareUSD:        1.575,
		EarlyMorningMinFareUSD: 1.80,
		Polygon: []LatLngPoint{
			{Lat: 10.12, Lng: -66.85},
			{Lat: 10.13, Lng: -66.85},
			{Lat: 10.13, Lng: -66.84},
		},
	}
}

func TestZoneFareRuleRequest_AcceptsValidPolygon(t *testing.T) {
	req := validZoneFareRuleRequest()
	if err := req.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// "dive" es obligatorio en el tag de Polygon: sin él, el validador no entra a
// los puntos del slice y una latitud/longitud fuera de rango pasaría.
func TestZoneFareRuleRequest_RejectsInvalidPolygonPoint(t *testing.T) {
	req := validZoneFareRuleRequest()
	req.Polygon[1].Lat = 999

	if err := req.Validate(); err == nil {
		t.Fatal("expected error for out-of-range latitude inside the polygon")
	}
}

func TestZoneFareRuleRequest_RejectsTooFewPoints(t *testing.T) {
	req := validZoneFareRuleRequest()
	req.Polygon = req.Polygon[:2]

	if err := req.Validate(); err == nil {
		t.Fatal("expected error for a polygon with fewer than 3 points")
	}
}
