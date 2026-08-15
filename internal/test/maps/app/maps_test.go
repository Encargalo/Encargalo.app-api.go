package app_test

import (
	"context"
	"errors"
	"testing"

	"Encargalo.app-api.go/internal/maps/app"
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
