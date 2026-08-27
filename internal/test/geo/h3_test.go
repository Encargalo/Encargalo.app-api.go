package geo_test

import (
	"slices"
	"testing"

	"Encargalo.app-api.go/internal/pkg/geo"
)

func TestLatLngToCell_SamePointSameCell(t *testing.T) {
	lat, lng := 10.4806, -66.9036 // Caracas

	first, err := geo.LatLngToCell(lat, lng, geo.ZoneAnalyticsResolution)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	second, err := geo.LatLngToCell(lat, lng, geo.ZoneAnalyticsResolution)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if first != second {
		t.Fatalf("expected same cell for same point, got %q and %q", first, second)
	}
}

func TestLatLngToCell_DifferentResolutionsDifferentCells(t *testing.T) {
	lat, lng := 10.4806, -66.9036

	coarse, err := geo.LatLngToCell(lat, lng, geo.ZoneAnalyticsResolution)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	fine, err := geo.LatLngToCell(lat, lng, geo.DriverMatchResolution)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if coarse == fine {
		t.Fatalf("expected different cells at different resolutions, got %q for both", coarse)
	}
}

func TestCellToLatLng_RoundTripsNearOriginalPoint(t *testing.T) {
	lat, lng := 10.4806, -66.9036

	cell, err := geo.LatLngToCell(lat, lng, geo.DriverMatchResolution)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	centerLat, centerLng, err := geo.CellToLatLng(cell)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// El centro de la celda debe quedar cerca del punto original (celda de
	// resolución 8, ~0.46 km²), tolerancia chica en grados.
	const tolerance = 0.01
	if diff := centerLat - lat; diff > tolerance || diff < -tolerance {
		t.Fatalf("latitud del centro %v demasiado lejos del punto original %v", centerLat, lat)
	}
	if diff := centerLng - lng; diff > tolerance || diff < -tolerance {
		t.Fatalf("longitud del centro %v demasiado lejos del punto original %v", centerLng, lng)
	}
}

func TestCellToLatLng_InvalidCellReturnsError(t *testing.T) {
	if _, _, err := geo.CellToLatLng("not-a-cell"); err == nil {
		t.Fatal("expected error for invalid cell, got nil")
	}
}

func TestGridDisk_KZeroReturnsOnlyOriginalCell(t *testing.T) {
	cell, err := geo.LatLngToCell(10.4806, -66.9036, geo.DriverMatchResolution)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	disk, err := geo.GridDisk(cell, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(disk) != 1 || disk[0] != cell {
		t.Fatalf("expected GridDisk(k=0) to return only [%q], got %v", cell, disk)
	}
}

func TestGridDisk_KOneIncludesOriginalAndSixNeighbors(t *testing.T) {
	cell, err := geo.LatLngToCell(10.4806, -66.9036, geo.DriverMatchResolution)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	disk, err := geo.GridDisk(cell, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Una celda hexagonal tiene 6 vecinos: el anillo k=1 trae la celda
	// original + hasta 6 vecinas (7 en el caso general, no-pentagonal).
	if len(disk) != 7 {
		t.Fatalf("expected 7 cells in GridDisk(k=1) for a hexagonal cell, got %d: %v", len(disk), disk)
	}

	if !slices.Contains(disk, cell) {
		t.Fatalf("expected original cell %q to be included in its own GridDisk", cell)
	}
}

func TestGridDisk_InvalidCellReturnsError(t *testing.T) {
	if _, err := geo.GridDisk("not-a-cell", 1); err == nil {
		t.Fatal("expected error for invalid cell, got nil")
	}
}
