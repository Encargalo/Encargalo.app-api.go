package app_test

import (
	"context"
	"testing"

	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Bogotá: coordenadas válidas a nivel global pero fuera del área de servicio.
const (
	outsideServiceAreaLat = 4.60971
	outsideServiceAreaLng = -74.08175
)

func TestEstimateFare_PickupOutsideVenezuela(t *testing.T) {
	svc, _, _, bcv, quotes := newTripsApp(t)
	req := sampleFareEstimateRequest(uuid.New())
	req.PickupLatitude, req.PickupLongitude = outsideServiceAreaLat, outsideServiceAreaLng

	_, err := svc.EstimateFare(context.Background(), req)

	require.ErrorIs(t, err, errcustom.ErrInvalidCoordinates)
	bcv.AssertNotCalled(t, "GetRate", mock.Anything)
	quotes.AssertNotCalled(t, "Save", mock.Anything, mock.Anything, mock.Anything)
}

func TestEstimateFare_DropoffOutsideVenezuela(t *testing.T) {
	svc, _, _, bcv, quotes := newTripsApp(t)
	req := sampleFareEstimateRequest(uuid.New())
	req.DropoffLatitude, req.DropoffLongitude = outsideServiceAreaLat, outsideServiceAreaLng

	_, err := svc.EstimateFare(context.Background(), req)

	require.ErrorIs(t, err, errcustom.ErrInvalidCoordinates)
	bcv.AssertNotCalled(t, "GetRate", mock.Anything)
	quotes.AssertNotCalled(t, "Save", mock.Anything, mock.Anything, mock.Anything)
}

func TestNearbyRiders_OutsideVenezuela(t *testing.T) {
	svc, _, _, _, _ := newTripsApp(t)

	_, err := svc.NearbyRiders(outsideServiceAreaLat, outsideServiceAreaLng)

	require.ErrorIs(t, err, errcustom.ErrInvalidCoordinates)
}
