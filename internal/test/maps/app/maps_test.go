package app_test

import (
	"context"
	"errors"
	"testing"

	"Encargalo.app-api.go/internal/maps/app"
	"Encargalo.app-api.go/internal/maps/domain/dtos"
	"Encargalo.app-api.go/internal/maps/domain/models"
	mocksmaps "Encargalo.app-api.go/internal/mocks/maps/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestReverseGeocode_InvalidCoordinates(t *testing.T) {
	repo := mocksmaps.NewMapsRepo(t)
	mapsApp := app.New(repo)

	_, err := mapsApp.ReverseGeocode(context.Background(), 999, -74.08175)

	require.ErrorIs(t, err, errcustom.ErrInvalidCoordinates)
	repo.AssertNotCalled(t, "ReverseGeocode", mock.Anything, mock.Anything, mock.Anything)
}

func TestReverseGeocode_ValidCoordinates(t *testing.T) {
	repo := mocksmaps.NewMapsRepo(t)
	mapsApp := app.New(repo)

	result := &models.ReverseGeocodeResult{FormattedAddress: "Calle 100, Bogotá"}
	repo.On("ReverseGeocode", context.Background(), 4.60971, -74.08175).Return(result, nil)

	got, err := mapsApp.ReverseGeocode(context.Background(), 4.60971, -74.08175)

	require.NoError(t, err)
	require.Equal(t, result, got)
}

func TestAutocompletePlaces_QueryTooShort(t *testing.T) {
	repo := mocksmaps.NewMapsRepo(t)
	mapsApp := app.New(repo)

	_, err := mapsApp.AutocompletePlaces(context.Background(), "  ab  ", nil, nil, "session-1")

	require.ErrorIs(t, err, errcustom.ErrInvalidSearchQuery)
	repo.AssertNotCalled(t, "AutocompletePlaces", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestAutocompletePlaces_InvalidBiasCoordinates(t *testing.T) {
	repo := mocksmaps.NewMapsRepo(t)
	mapsApp := app.New(repo)

	badLat := 999.0
	badLng := -74.08175

	_, err := mapsApp.AutocompletePlaces(context.Background(), "Calle 100", &badLat, &badLng, "session-1")

	require.ErrorIs(t, err, errcustom.ErrInvalidCoordinates)
	repo.AssertNotCalled(t, "AutocompletePlaces", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestAutocompletePlaces_TrimsQueryBeforeCallingRepo(t *testing.T) {
	repo := mocksmaps.NewMapsRepo(t)
	mapsApp := app.New(repo)

	predictions := []models.PlacePrediction{{PlaceID: "place-1"}}
	repo.On("AutocompletePlaces", context.Background(), "Calle 100", (*float64)(nil), (*float64)(nil), "session-1").Return(predictions, nil)

	got, err := mapsApp.AutocompletePlaces(context.Background(), "  Calle 100  ", nil, nil, "session-1")

	require.NoError(t, err)
	require.Equal(t, predictions, got)
}

func TestGetPlaceDetails_PassthroughError(t *testing.T) {
	repo := mocksmaps.NewMapsRepo(t)
	mapsApp := app.New(repo)

	repo.On("GetPlaceDetails", context.Background(), "place-1", "session-1").Return(nil, errors.New("google timeout"))

	_, err := mapsApp.GetPlaceDetails(context.Background(), "place-1", "session-1")

	require.EqualError(t, err, "google timeout")
}

func TestDirections_InvalidCoordinates(t *testing.T) {
	repo := mocksmaps.NewMapsRepo(t)
	mapsApp := app.New(repo)

	_, err := mapsApp.Directions(context.Background(), 10.48, -66.90, 95, -66.90)

	require.ErrorIs(t, err, errcustom.ErrInvalidCoordinates)
	repo.AssertNotCalled(t, "Directions", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestDirections_DelegatesToRepo(t *testing.T) {
	repo := mocksmaps.NewMapsRepo(t)
	mapsApp := app.New(repo)

	route := &models.Route{Polyline: "abc", DistanceMeters: 1234, DurationMin: 5}
	repo.On("Directions", mock.Anything, 10.48, -66.90, 10.49, -66.88).Return(route, nil)

	got, err := mapsApp.Directions(context.Background(), 10.48, -66.90, 10.49, -66.88)

	require.NoError(t, err)
	require.Equal(t, route, got)
}

func TestPopularPlaces_UsesTheFixedRadius(t *testing.T) {
	repo := mocksmaps.NewMapsRepo(t)
	mapsApp := app.New(repo)

	repo.On("PopularPlaces", context.Background(), 10.17809, -66.81615, 10000).Return([]models.PopularPlace{}, nil)

	_, err := mapsApp.PopularPlaces(context.Background(), dtos.PopularPlacesQuery{Latitude: 10.17809, Longitude: -66.81615})

	require.NoError(t, err)
}

func TestPopularPlaces_SortsByDistanceAndCalculatesIt(t *testing.T) {
	repo := mocksmaps.NewMapsRepo(t)
	mapsApp := app.New(repo)

	repo.On("PopularPlaces", context.Background(), 10.17809, -66.81615, 10000).Return([]models.PopularPlace{
		{PlaceID: "lejos", Lat: 10.2370073, Lng: -66.8579879},
		{PlaceID: "cerca", Lat: 10.18, Lng: -66.816},
	}, nil)

	got, err := mapsApp.PopularPlaces(context.Background(), dtos.PopularPlacesQuery{Latitude: 10.17809, Longitude: -66.81615})

	require.NoError(t, err)
	require.Len(t, got, 2)
	require.Equal(t, "cerca", got[0].PlaceID)
	require.Equal(t, "lejos", got[1].PlaceID)
	require.Greater(t, got[1].DistanceKm, got[0].DistanceKm)
	require.InDelta(t, 7.99, got[1].DistanceKm, 0.05)
}

func TestPopularPlaces_TruncatesToLimit(t *testing.T) {
	repo := mocksmaps.NewMapsRepo(t)
	mapsApp := app.New(repo)

	repo.On("PopularPlaces", context.Background(), 10.17809, -66.81615, 10000).Return([]models.PopularPlace{
		{PlaceID: "a", Lat: 10.179, Lng: -66.816},
		{PlaceID: "b", Lat: 10.185, Lng: -66.816},
		{PlaceID: "c", Lat: 10.19, Lng: -66.816},
	}, nil)

	got, err := mapsApp.PopularPlaces(context.Background(), dtos.PopularPlacesQuery{Latitude: 10.17809, Longitude: -66.81615, Limit: 2})

	require.NoError(t, err)
	require.Len(t, got, 2)
	require.Equal(t, "a", got[0].PlaceID)
}

func TestPopularPlaces_PassthroughError(t *testing.T) {
	repo := mocksmaps.NewMapsRepo(t)
	mapsApp := app.New(repo)

	repo.On("PopularPlaces", context.Background(), 10.17809, -66.81615, 10000).Return(nil, errcustom.ErrMapsUnavailable)

	_, err := mapsApp.PopularPlaces(context.Background(), dtos.PopularPlacesQuery{Latitude: 10.17809, Longitude: -66.81615})

	require.ErrorIs(t, err, errcustom.ErrMapsUnavailable)
}

func TestPopularPlaces_RejectsCoordinatesOutOfRange(t *testing.T) {
	repo := mocksmaps.NewMapsRepo(t)
	mapsApp := app.New(repo)

	_, err := mapsApp.PopularPlaces(context.Background(), dtos.PopularPlacesQuery{Latitude: 999, Longitude: -66.81615})

	require.ErrorIs(t, err, errcustom.ErrInvalidCoordinates)
}

func TestPopularPlaces_ClampsLimitToTheMaximum(t *testing.T) {
	repo := mocksmaps.NewMapsRepo(t)
	mapsApp := app.New(repo)

	repo.On("PopularPlaces", context.Background(), 10.17809, -66.81615, 10000).Return([]models.PopularPlace{
		{PlaceID: "a", Lat: 10.179, Lng: -66.816},
		{PlaceID: "b", Lat: 10.185, Lng: -66.816},
	}, nil)

	got, err := mapsApp.PopularPlaces(context.Background(), dtos.PopularPlacesQuery{Latitude: 10.17809, Longitude: -66.81615, Limit: 999})

	require.NoError(t, err)
	require.Len(t, got, 2)
}

func TestPopularPlaces_DropsPlacesOutsideTheRadius(t *testing.T) {
	repo := mocksmaps.NewMapsRepo(t)
	mapsApp := app.New(repo)

	// El segundo lugar está a ~11.5 km del usuario: entra en el radio expandido que consulta el repo, pero no en los 10 km del contrato.
	repo.On("PopularPlaces", context.Background(), 10.17809, -66.81615, 10000).Return([]models.PopularPlace{
		{PlaceID: "dentro", Lat: 10.18, Lng: -66.816},
		{PlaceID: "fuera", Lat: 10.28179, Lng: -66.81615},
	}, nil)

	got, err := mapsApp.PopularPlaces(context.Background(), dtos.PopularPlacesQuery{Latitude: 10.17809, Longitude: -66.81615})

	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, "dentro", got[0].PlaceID)
}
