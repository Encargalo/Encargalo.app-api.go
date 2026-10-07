package handler_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	mapsmodels "Encargalo.app-api.go/internal/maps/domain/models"
	mapshandler "Encargalo.app-api.go/internal/maps/handler"
	mocksmaps "Encargalo.app-api.go/internal/mocks/maps/domain/ports"
	"Encargalo.app-api.go/internal/shared/buildhttptest"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func httpCode(err error) int {
	if he, ok := err.(*echo.HTTPError); ok {
		return he.Code
	}
	return 0
}

func newHandler(t *testing.T) (mapshandler.MapsHandler, *mocksmaps.MapsApp) {
	t.Helper()
	svc := mocksmaps.NewMapsApp(t)
	return mapshandler.NewMapsHandler(svc), svc
}

func TestReverseGeocode_Success(t *testing.T) {
	h, svc := newHandler(t)

	result := &mapsmodels.ReverseGeocodeResult{
		FormattedAddress: "Calle 100 #10-20, Bogotá",
		ShortAddress:     "Calle 100",
	}
	tc := buildhttptest.SetupControllerCase("/maps/geocode/reverse?lat=4.60971&lng=-74.08175", http.MethodGet, nil)

	svc.On("ReverseGeocode", tc.Req.Context(), 4.60971, -74.08175).Return(result, nil)

	err := h.ReverseGeocode(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)

	var got mapsmodels.ReverseGeocodeResult
	assert.NoError(t, json.Unmarshal(tc.Res.Body.Bytes(), &got))
	assert.Equal(t, result.FormattedAddress, got.FormattedAddress)
}

func TestReverseGeocode_MissingLat(t *testing.T) {
	h, _ := newHandler(t)

	tc := buildhttptest.SetupControllerCase("/maps/geocode/reverse?lng=-74.08175", http.MethodGet, nil)

	err := h.ReverseGeocode(tc.Ctx)

	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestReverseGeocode_AppError(t *testing.T) {
	h, svc := newHandler(t)

	tc := buildhttptest.SetupControllerCase("/maps/geocode/reverse?lat=4.60971&lng=-74.08175", http.MethodGet, nil)

	svc.On("ReverseGeocode", tc.Req.Context(), 4.60971, -74.08175).Return(nil, errors.New("google timeout"))

	err := h.ReverseGeocode(tc.Ctx)

	assert.EqualError(t, err, "google timeout")
}

func TestReverseGeocode_NotFoundPropagatesFromApp(t *testing.T) {
	h, svc := newHandler(t)

	tc := buildhttptest.SetupControllerCase("/maps/geocode/reverse?lat=4.60971&lng=-74.08175", http.MethodGet, nil)

	svc.On("ReverseGeocode", tc.Req.Context(), 4.60971, -74.08175).Return(nil, errcustom.ErrNotFound)

	err := h.ReverseGeocode(tc.Ctx)

	assert.ErrorIs(t, err, errcustom.ErrNotFound)
}

func TestAutocompletePlaces_Success(t *testing.T) {
	h, svc := newHandler(t)

	predictions := []mapsmodels.PlacePrediction{
		{PlaceID: "place-1", MainText: "Calle 100", SecondaryText: "Bogotá"},
	}
	tc := buildhttptest.SetupControllerCase("/maps/places/autocomplete?query=Calle+100&session_token=abc-123", http.MethodGet, nil)

	svc.On("AutocompletePlaces", tc.Req.Context(), "Calle 100", (*float64)(nil), (*float64)(nil), "abc-123").Return(predictions, nil)

	err := h.AutocompletePlaces(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)

	var got []mapsmodels.PlacePrediction
	assert.NoError(t, json.Unmarshal(tc.Res.Body.Bytes(), &got))
	assert.Len(t, got, 1)
	assert.Equal(t, "place-1", got[0].PlaceID)
}

func TestAutocompletePlaces_WithBiasCoords(t *testing.T) {
	h, svc := newHandler(t)

	tc := buildhttptest.SetupControllerCase("/maps/places/autocomplete?query=Calle+100&lat=4.60971&lng=-74.08175&session_token=abc-123", http.MethodGet, nil)

	svc.On("AutocompletePlaces", tc.Req.Context(), "Calle 100", mock.MatchedBy(func(v *float64) bool { return v != nil && *v == 4.60971 }), mock.MatchedBy(func(v *float64) bool { return v != nil && *v == -74.08175 }), "abc-123").
		Return([]mapsmodels.PlacePrediction{}, nil)

	err := h.AutocompletePlaces(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
}

func TestAutocompletePlaces_MissingSessionToken(t *testing.T) {
	h, _ := newHandler(t)

	tc := buildhttptest.SetupControllerCase("/maps/places/autocomplete?query=Calle+100", http.MethodGet, nil)

	err := h.AutocompletePlaces(tc.Ctx)

	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

// La validación de longitud mínima de query vive en el app layer (ver
// internal/test/maps/app/maps_test.go) — acá solo confirmamos que el
// handler la propaga tal cual, sin reinterpretarla.
func TestAutocompletePlaces_QueryTooShortPropagatesFromApp(t *testing.T) {
	h, svc := newHandler(t)

	tc := buildhttptest.SetupControllerCase("/maps/places/autocomplete?query=ab&session_token=abc-123", http.MethodGet, nil)

	svc.On("AutocompletePlaces", tc.Req.Context(), "ab", (*float64)(nil), (*float64)(nil), "abc-123").Return(nil, errcustom.ErrInvalidSearchQuery)

	err := h.AutocompletePlaces(tc.Ctx)

	assert.ErrorIs(t, err, errcustom.ErrInvalidSearchQuery)
}

func TestGetPlaceDetails_Success(t *testing.T) {
	h, svc := newHandler(t)

	details := &mapsmodels.PlaceDetails{
		Address: "Calle 100 #10-20, Bogotá",
		Lat:     4.60971,
		Lng:     -74.08175,
	}
	tc := buildhttptest.SetupControllerCase("/maps/places/details?place_id=place-1&session_token=abc-123", http.MethodGet, nil)

	svc.On("GetPlaceDetails", tc.Req.Context(), "place-1", "abc-123").Return(details, nil)

	err := h.GetPlaceDetails(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)

	var got mapsmodels.PlaceDetails
	assert.NoError(t, json.Unmarshal(tc.Res.Body.Bytes(), &got))
	assert.Equal(t, details.Address, got.Address)
}

func TestGetPlaceDetails_MissingPlaceID(t *testing.T) {
	h, _ := newHandler(t)

	tc := buildhttptest.SetupControllerCase("/maps/places/details?session_token=abc-123", http.MethodGet, nil)

	err := h.GetPlaceDetails(tc.Ctx)

	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestGetPlaceDetails_MissingSessionToken(t *testing.T) {
	h, _ := newHandler(t)

	tc := buildhttptest.SetupControllerCase("/maps/places/details?place_id=place-1", http.MethodGet, nil)

	err := h.GetPlaceDetails(tc.Ctx)

	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestGetPlaceDetails_AppError(t *testing.T) {
	h, svc := newHandler(t)

	tc := buildhttptest.SetupControllerCase("/maps/places/details?place_id=place-1&session_token=abc-123", http.MethodGet, nil)

	svc.On("GetPlaceDetails", tc.Req.Context(), "place-1", "abc-123").Return(nil, errors.New("google timeout"))

	err := h.GetPlaceDetails(tc.Ctx)

	assert.EqualError(t, err, "google timeout")
}

func TestGetPlaceDetails_NotFoundPropagatesFromApp(t *testing.T) {
	h, svc := newHandler(t)

	tc := buildhttptest.SetupControllerCase("/maps/places/details?place_id=place-1&session_token=abc-123", http.MethodGet, nil)

	svc.On("GetPlaceDetails", tc.Req.Context(), "place-1", "abc-123").Return(nil, errcustom.ErrNotFound)

	err := h.GetPlaceDetails(tc.Ctx)

	assert.ErrorIs(t, err, errcustom.ErrNotFound)
}

func TestDirections_Success(t *testing.T) {
	h, svc := newHandler(t)

	route := &mapsmodels.Route{Polyline: "abc", DistanceMeters: 1234, DurationMin: 5}
	tc := buildhttptest.SetupControllerCase("/maps/directions?origin_lat=10.48&origin_lng=-66.9&dest_lat=10.49&dest_lng=-66.88", http.MethodGet, nil)
	svc.On("Directions", tc.Req.Context(), 10.48, -66.9, 10.49, -66.88).Return(route, nil)

	err := h.Directions(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
	var got mapsmodels.Route
	assert.NoError(t, json.Unmarshal(tc.Res.Body.Bytes(), &got))
	assert.Equal(t, *route, got)
}

func TestDirections_MissingCoordinate(t *testing.T) {
	h, _ := newHandler(t)
	tc := buildhttptest.SetupControllerCase("/maps/directions?origin_lat=10.48&origin_lng=-66.9&dest_lat=10.49", http.MethodGet, nil)

	err := h.Directions(tc.Ctx)

	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestDirections_ErrorsMapToContractStatus(t *testing.T) {
	cases := map[error]int{
		errcustom.ErrNotFound:           http.StatusNotFound,
		errcustom.ErrMapsUnavailable:    http.StatusBadGateway,
		errcustom.ErrInvalidCoordinates: http.StatusBadRequest,
	}
	for appErr, status := range cases {
		h, svc := newHandler(t)
		tc := buildhttptest.SetupControllerCase("/maps/directions?origin_lat=10.48&origin_lng=-66.9&dest_lat=10.49&dest_lng=-66.88", http.MethodGet, nil)
		svc.On("Directions", tc.Req.Context(), 10.48, -66.9, 10.49, -66.88).Return(nil, appErr)

		err := h.Directions(tc.Ctx)

		assert.ErrorIs(t, err, appErr)
		assert.Equal(t, status, errcustom.HTTPStatus(err))
	}
}
