package app

import (
	"context"
	"strings"
	"unicode/utf8"

	"Encargalo.app-api.go/internal/maps/domain/models"
	"Encargalo.app-api.go/internal/maps/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
)

// minSearchQueryRunes evita que una búsqueda trivial (1-2 caracteres) le
// cueste dinero al backend en cada tecla — el móvil ya aplica el mismo
// mínimo por UX, pero el servidor no puede confiar en eso.
const minSearchQueryRunes = 3

type mapsApp struct {
	repo ports.MapsRepo
}

func New(repo ports.MapsRepo) ports.MapsApp {
	return &mapsApp{repo}
}

func (a *mapsApp) ReverseGeocode(ctx context.Context, lat, lng float64) (*models.ReverseGeocodeResult, error) {
	if !validCoordinates(lat, lng) {
		return nil, errcustom.ErrInvalidCoordinates
	}
	return a.repo.ReverseGeocode(ctx, lat, lng)
}

func (a *mapsApp) AutocompletePlaces(ctx context.Context, query string, biasLat, biasLng *float64, sessionToken string) ([]models.PlacePrediction, error) {
	query = strings.TrimSpace(query)
	if utf8.RuneCountInString(query) < minSearchQueryRunes {
		return nil, errcustom.ErrInvalidSearchQuery
	}
	if biasLat != nil && biasLng != nil && !validCoordinates(*biasLat, *biasLng) {
		return nil, errcustom.ErrInvalidCoordinates
	}
	return a.repo.AutocompletePlaces(ctx, query, biasLat, biasLng, sessionToken)
}

func (a *mapsApp) GetPlaceDetails(ctx context.Context, placeID, sessionToken string) (*models.PlaceDetails, error) {
	return a.repo.GetPlaceDetails(ctx, placeID, sessionToken)
}

func (a *mapsApp) Directions(ctx context.Context, originLat, originLng, destLat, destLng float64) (*models.Route, error) {
	if !validCoordinates(originLat, originLng) || !validCoordinates(destLat, destLng) {
		return nil, errcustom.ErrInvalidCoordinates
	}
	return a.repo.Directions(ctx, originLat, originLng, destLat, destLng)
}

func validCoordinates(lat, lng float64) bool {
	return lat >= -90 && lat <= 90 && lng >= -180 && lng <= 180
}
