package ports

import (
	"context"

	"Encargalo.app-api.go/internal/maps/domain/dtos"
	"Encargalo.app-api.go/internal/maps/domain/models"
)

type MapsApp interface {
	ReverseGeocode(ctx context.Context, lat, lng float64) (*models.ReverseGeocodeResult, error)
	AutocompletePlaces(ctx context.Context, query string, biasLat, biasLng *float64, sessionToken string) ([]models.PlacePrediction, error)
	GetPlaceDetails(ctx context.Context, placeID, sessionToken string) (*models.PlaceDetails, error)
	Directions(ctx context.Context, originLat, originLng, destLat, destLng float64) (*models.Route, error)
	PopularPlaces(ctx context.Context, query dtos.PopularPlacesQuery) ([]models.PopularPlace, error)
}

type MapsRepo interface {
	ReverseGeocode(ctx context.Context, lat, lng float64) (*models.ReverseGeocodeResult, error)
	AutocompletePlaces(ctx context.Context, query string, biasLat, biasLng *float64, sessionToken string) ([]models.PlacePrediction, error)
	GetPlaceDetails(ctx context.Context, placeID, sessionToken string) (*models.PlaceDetails, error)
	Directions(ctx context.Context, originLat, originLng, destLat, destLng float64) (*models.Route, error)
	PopularPlaces(ctx context.Context, lat, lng float64, radiusMeters int) ([]models.PopularPlace, error)
}
