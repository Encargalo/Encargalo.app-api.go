package geo

import (
	"math"
	"strings"
)

// EncodePolyline codifica puntos [lat, lng] con el Encoded Polyline Algorithm de Google.
func EncodePolyline(points [][2]float64) string {
	var b strings.Builder
	prevLat, prevLng := 0, 0
	for _, p := range points {
		lat := int(math.Round(p[0] * 1e5))
		lng := int(math.Round(p[1] * 1e5))
		encodePolylineValue(&b, lat-prevLat)
		encodePolylineValue(&b, lng-prevLng)
		prevLat, prevLng = lat, lng
	}
	return b.String()
}

func encodePolylineValue(b *strings.Builder, v int) {
	shifted := v << 1
	if v < 0 {
		shifted = ^shifted
	}
	for shifted >= 0x20 {
		b.WriteByte(byte((0x20 | (shifted & 0x1f)) + 63))
		shifted >>= 5
	}
	b.WriteByte(byte(shifted + 63))
}
