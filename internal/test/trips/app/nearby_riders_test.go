package app_test

import (
	"context"
	"testing"

	mockstrips "Encargalo.app-api.go/internal/mocks/trips/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/trips/app"
	"Encargalo.app-api.go/internal/trips/domain/dtos"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNearbyRiders_InvalidCoordinates(t *testing.T) {
	svc := app.NewTripsApp(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	_, err := svc.NearbyRiders(context.Background(), 120, 0)

	require.ErrorIs(t, err, errcustom.ErrInvalidCoordinates)
}

func TestNearbyRiders_EmptyWhenLookupMissing(t *testing.T) {
	svc := app.NewTripsApp(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	resp, err := svc.NearbyRiders(context.Background(), 10.48, -66.90)

	require.NoError(t, err)
	require.Empty(t, resp.Riders)
}

func TestNearbyRiders_ReturnsLookupPins(t *testing.T) {
	nearby := mockstrips.NewNearbyRidersLookup(t)
	svc := app.NewTripsApp(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nearby, nil, nil, nil, nil)
	pins := []dtos.NearbyRider{{Latitude: 10.481, Longitude: -66.904}}
	nearby.On("NearbyRiders", mock.Anything, 10.48, -66.90).Return(pins, nil)

	resp, err := svc.NearbyRiders(context.Background(), 10.48, -66.90)

	require.NoError(t, err)
	require.Equal(t, pins, resp.Riders)
}

func TestNearbyRiders_PropagatesLookupError(t *testing.T) {
	nearby := mockstrips.NewNearbyRidersLookup(t)
	svc := app.NewTripsApp(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nearby, nil, nil, nil, nil)
	nearby.On("NearbyRiders", mock.Anything, 10.48, -66.90).Return(nil, errcustom.ErrUnexpectedError)

	_, err := svc.NearbyRiders(context.Background(), 10.48, -66.90)

	require.ErrorIs(t, err, errcustom.ErrUnexpectedError)
}
