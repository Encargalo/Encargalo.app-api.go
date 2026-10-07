package geo_test

import (
	"testing"

	"Encargalo.app-api.go/internal/pkg/geo"
	"github.com/stretchr/testify/assert"
)

func TestEncodePolyline_GoogleReferenceExample(t *testing.T) {
	// Ejemplo de la documentación del Encoded Polyline Algorithm de Google.
	points := [][2]float64{{38.5, -120.2}, {40.7, -120.95}, {43.252, -126.453}}

	assert.Equal(t, "_p~iF~ps|U_ulLnnqC_mqNvxq`@", geo.EncodePolyline(points))
}

func TestEncodePolyline_Empty(t *testing.T) {
	assert.Empty(t, geo.EncodePolyline(nil))
}
