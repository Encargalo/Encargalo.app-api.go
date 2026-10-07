package repo_test

import (
	"context"
	"testing"

	"Encargalo.app-api.go/internal/maps/repo"
	"Encargalo.app-api.go/internal/pkg/geo"
	"github.com/stretchr/testify/require"
)

func TestDirections_OutsideProductionReturnsStraightLine(t *testing.T) {
	r := repo.NewMapsRepo(nil)

	route, err := r.Directions(context.Background(), 10.4941, -66.8919, 10.4989, -66.8807)

	require.NoError(t, err)
	require.Equal(t, geo.EncodePolyline([][2]float64{{10.4941, -66.8919}, {10.4989, -66.8807}}), route.Polyline)
	require.InDelta(t, 1340, route.DistanceMeters, 20)
	require.Zero(t, route.DurationMin)
}
