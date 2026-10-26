package geo

import (
	"fmt"

	"github.com/uber/h3-go/v4"
)

const (
	// DriverMatchResolution es la resolución H3 para bucketing de conductores
	// en tiempo real (celdas de ~0.74 km²) — lo bastante fina para agrupar
	// candidatos cercanos sin escanear todos los conductores conectados.
	DriverMatchResolution = 8

	// ZoneAnalyticsResolution es la resolución H3 para agregación de zonas en
	// analítica/heatmaps (celdas de ~5.16 km²) — más gruesa que la de
	// matching para no fragmentar demasiado el heatmap.
	ZoneAnalyticsResolution = 7
)

// LatLngToCell indexa un punto lat/lng en la celda H3 de la resolución dada.
func LatLngToCell(lat, lng float64, res int) (string, error) {
	cell, err := h3.LatLngToCell(h3.NewLatLng(lat, lng), res)
	if err != nil {
		return "", err
	}
	return cell.String(), nil
}

// CellToLatLng devuelve el centro de la celda H3 dada.
func CellToLatLng(cell string) (lat, lng float64, err error) {
	c := h3.CellFromString(cell)
	if !c.IsValid() {
		return 0, 0, errInvalidCell(cell)
	}
	latLng, err := h3.CellToLatLng(c)
	if err != nil {
		return 0, 0, err
	}
	return latLng.Lat, latLng.Lng, nil
}

// GridDisk devuelve la celda dada más todas sus vecinas hasta k anillos
// (k=0 devuelve solo la celda original).
func GridDisk(cell string, k int) ([]string, error) {
	c := h3.CellFromString(cell)
	if !c.IsValid() {
		return nil, errInvalidCell(cell)
	}
	cells, err := h3.GridDisk(c, k)
	if err != nil {
		return nil, err
	}
	out := make([]string, len(cells))
	for i, cc := range cells {
		out[i] = cc.String()
	}
	return out, nil
}

// h3Res8NeighborKm es la distancia entre centros de celdas vecinas en resolución 8: la arista por raíz de 3.
const h3Res8NeighborKm = 0.7991

// RingsToKm traduce k anillos de GridDisk al radio en km que cubren; el medio
// anillo extra es el apotema de la celda del borde, no su centro.
func RingsToKm(rings int) float64 {
	if rings < 0 {
		return 0
	}
	return (float64(rings) + 0.5) * h3Res8NeighborKm
}

func errInvalidCell(cell string) error {
	return fmt.Errorf("geo: celda H3 inválida: %q", cell)
}
