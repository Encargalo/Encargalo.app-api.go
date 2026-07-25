package geo_test

import (
	"testing"

	"Encargalo.app-api.go/internal/pkg/geo"
	"github.com/stretchr/testify/assert"
)

func TestHaversineKm(t *testing.T) {
	// Caracas: Plaza Venezuela (10.4941,-66.8919) vs Sabana Grande (10.4989,-66.8807),
	// ~1.34 km en línea recta.
	distance := geo.HaversineKm(10.4941, -66.8919, 10.4989, -66.8807)
	assert.InDelta(t, 1.34, distance, 0.1)
}

func TestHaversineKm_SamePoint(t *testing.T) {
	distance := geo.HaversineKm(10.4806, -66.9036, 10.4806, -66.9036)
	assert.InDelta(t, 0, distance, 0.0001)
}
