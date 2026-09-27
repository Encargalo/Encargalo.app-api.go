package handler_test

import (
	"net/http"
	"testing"

	"Encargalo.app-api.go/internal/shared/buildhttptest"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/trips/domain/dtos"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNearbyRiders_MissingCoordinates(t *testing.T) {
	h, _ := newTripsHandler(t)
	tc := buildhttptest.SetupControllerCase("/trips/nearby-riders", http.MethodGet, nil)

	err := h.NearbyRiders(tc.Ctx)

	assert.Equal(t, http.StatusBadRequest, nearbyHTTPCode(err))
}

func TestNearbyRiders_InvalidCoordinates(t *testing.T) {
	h, _ := newTripsHandler(t)
	tc := buildhttptest.SetupControllerCase("/trips/nearby-riders?lat=no&lng=-66.9", http.MethodGet, nil)

	err := h.NearbyRiders(tc.Ctx)

	assert.Equal(t, http.StatusBadRequest, nearbyHTTPCode(err))
}

func TestNearbyRiders_Success(t *testing.T) {
	h, svc := newTripsHandler(t)
	tc := buildhttptest.SetupControllerCase("/trips/nearby-riders?lat=10.4806&lng=-66.9036", http.MethodGet, nil)
	svc.On("NearbyRiders", 10.4806, -66.9036).Return(dtos.NearbyRidersResponse{
		Riders: []dtos.NearbyRider{{Latitude: 10.481, Longitude: -66.904}},
	}, nil)

	err := h.NearbyRiders(tc.Ctx)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
	assert.Contains(t, tc.Res.Body.String(), `"latitude":10.481`)
	assert.NotContains(t, tc.Res.Body.String(), "rider_id")
}

func TestNearbyRiders_ServiceError(t *testing.T) {
	h, svc := newTripsHandler(t)
	tc := buildhttptest.SetupControllerCase("/trips/nearby-riders?lat=10.48&lng=-66.90", http.MethodGet, nil)
	svc.On("NearbyRiders", 10.48, -66.90).Return(dtos.NearbyRidersResponse{}, errcustom.ErrUnexpectedError)

	err := h.NearbyRiders(tc.Ctx)

	assert.Equal(t, http.StatusInternalServerError, nearbyHTTPCode(err))
}

func nearbyHTTPCode(err error) int {
	if he, ok := err.(*echo.HTTPError); ok {
		return he.Code
	}
	return 0
}
