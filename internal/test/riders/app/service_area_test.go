package app_test

import (
	"context"
	"testing"

	mocksriders "Encargalo.app-api.go/internal/mocks/riders/domain/ports"
	riderModels "Encargalo.app-api.go/internal/riders/domain/models"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// Bogotá: coordenadas válidas para H3 pero fuera del área de servicio.
const (
	outsideServiceAreaLat = 4.60971
	outsideServiceAreaLng = -74.08175
)

func TestUpdateLocation_OutsideVenezuela(t *testing.T) {
	svc := newNearbyMgr(t)
	riderID := uuid.New()
	require.NoError(t, svc.Register(context.Background(), riderID, mocksriders.NewRiderConn(t), riderModels.RiderModeTrips))

	err := svc.UpdateLocation(context.Background(), riderID, outsideServiceAreaLat, outsideServiceAreaLng)

	require.ErrorIs(t, err, errcustom.ErrInvalidCoordinates)
}

// Una ubicación rechazada no se guarda: el rider no puede aparecer en el mapa desde fuera del área de servicio.
func TestUpdateLocation_RejectedLocationIsNotStored(t *testing.T) {
	svc := newNearbyMgr(t)
	lat, lng := 10.4806, -66.9036
	riderID, _ := placeTripRider(t, svc, lat, lng)

	require.Error(t, svc.UpdateLocation(context.Background(), riderID, outsideServiceAreaLat, outsideServiceAreaLng))

	pins, err := asNearby(t, svc).NearbyRiders(lat, lng)
	require.NoError(t, err)
	require.Len(t, pins, 1)
	require.Equal(t, 10.481, pins[0].Latitude)
}

func TestNearbyRiders_LookupOutsideVenezuela(t *testing.T) {
	svc := newNearbyMgr(t)

	_, err := asNearby(t, svc).NearbyRiders(outsideServiceAreaLat, outsideServiceAreaLng)

	require.ErrorIs(t, err, errcustom.ErrInvalidCoordinates)
}
