package geo_test

import (
	"slices"
	"testing"

	"Encargalo.app-api.go/internal/pkg/geo"
)

func TestPolygonToCells_ContainsCellOfInteriorPoint(t *testing.T) {
	// Cuadrado alrededor de Caracas, lo bastante grande para cubrir
	// varias celdas de resolución 7 (~5.16 km² cada una).
	polygon := []geo.LatLng{
		{Lat: 10.40, Lng: -67.00},
		{Lat: 10.60, Lng: -67.00},
		{Lat: 10.60, Lng: -66.80},
		{Lat: 10.40, Lng: -66.80},
	}

	cells, err := geo.PolygonToCells(polygon, geo.ZoneAnalyticsResolution)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cells) == 0 {
		t.Fatal("expected al menos una celda para un polígono con área")
	}

	interiorCell, err := geo.LatLngToCell(10.4806, -66.9036, geo.ZoneAnalyticsResolution)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !slices.Contains(cells, interiorCell) {
		t.Fatalf("expected la celda del punto interior %q en el resultado, got %v", interiorCell, cells)
	}
}

func TestPolygonToCells_DifferentResolutionsDifferentCellCounts(t *testing.T) {
	polygon := []geo.LatLng{
		{Lat: 10.40, Lng: -67.00},
		{Lat: 10.60, Lng: -67.00},
		{Lat: 10.60, Lng: -66.80},
		{Lat: 10.40, Lng: -66.80},
	}

	coarse, err := geo.PolygonToCells(polygon, geo.ZoneAnalyticsResolution)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	fine, err := geo.PolygonToCells(polygon, geo.DriverMatchResolution)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Resolución más fina sobre la misma área produce más celdas, no menos.
	if len(fine) <= len(coarse) {
		t.Fatalf("expected más celdas en resolución fina (%d) que en gruesa (%d)", len(fine), len(coarse))
	}
}
