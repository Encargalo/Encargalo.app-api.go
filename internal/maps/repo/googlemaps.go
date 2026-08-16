package repo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"Encargalo.app-api.go/internal/maps/domain/models"
	"Encargalo.app-api.go/internal/maps/domain/ports"
	"Encargalo.app-api.go/internal/pkg/cache"
	"Encargalo.app-api.go/internal/shared/adapters/googlemaps"
	"Encargalo.app-api.go/internal/shared/errcustom"
)

const (
	reverseGeocodePrefix = "maps:reverse_geocode:"
	reverseGeocodeTTL    = 15 * time.Minute

	placeDetailsPrefix = "maps:place_details:"
	placeDetailsTTL    = 24 * time.Hour
)

type mapsRepo struct {
	cache cache.Cache
}

func NewMapsRepo(cache cache.Cache) ports.MapsRepo {
	return &mapsRepo{cache}
}

// ReverseGeocode is cached by coordinates rounded to 5 decimals (~1.1m),
// since dragging the map fires this repeatedly over near-identical points.
func (r *mapsRepo) ReverseGeocode(ctx context.Context, lat, lng float64) (*models.ReverseGeocodeResult, error) {
	cacheKey := fmt.Sprintf("%s%.5f,%.5f", reverseGeocodePrefix, lat, lng)

	cached, _ := r.cache.Get(ctx, cacheKey)
	if cached != "" {
		var result models.ReverseGeocodeResult
		if err := json.Unmarshal([]byte(cached), &result); err == nil {
			return &result, nil
		}
		slog.Error("maps: error al hacer unmarshal del reverse geocode cacheado", "key", cacheKey)
	}

	raw, err := googlemaps.ReverseGeocode(ctx, lat, lng)
	if err != nil {
		if errors.Is(err, googlemaps.ErrNoResults) {
			return nil, errcustom.ErrNotFound
		}
		return nil, err
	}

	result := &models.ReverseGeocodeResult{
		FormattedAddress: raw.FormattedAddress,
		ShortAddress:     raw.ShortAddress,
	}

	if err := r.cache.Set(ctx, cacheKey, result, reverseGeocodeTTL); err != nil {
		slog.Error("maps: error al guardar el reverse geocode en el cache", "error", err)
	}

	return result, nil
}

// AutocompletePlaces is never cached: it depends on live input text plus a
// per-search sessiontoken, so caching would not hit.
func (r *mapsRepo) AutocompletePlaces(ctx context.Context, query string, biasLat, biasLng *float64, sessionToken string) ([]models.PlacePrediction, error) {
	raw, err := googlemaps.AutocompletePlaces(ctx, query, biasLat, biasLng, sessionToken)
	if err != nil {
		return nil, err
	}

	predictions := make([]models.PlacePrediction, 0, len(raw))
	for _, p := range raw {
		predictions = append(predictions, models.PlacePrediction{
			PlaceID:       p.PlaceID,
			MainText:      p.MainText,
			SecondaryText: p.SecondaryText,
		})
	}

	return predictions, nil
}

// GetPlaceDetails is cached by place_id: a place's address/coordinates don't
// change between searches.
func (r *mapsRepo) GetPlaceDetails(ctx context.Context, placeID, sessionToken string) (*models.PlaceDetails, error) {
	cacheKey := placeDetailsPrefix + placeID

	cached, _ := r.cache.Get(ctx, cacheKey)
	if cached != "" {
		var details models.PlaceDetails
		if err := json.Unmarshal([]byte(cached), &details); err == nil {
			return &details, nil
		}
		slog.Error("maps: error al hacer unmarshal del place details cacheado", "key", cacheKey)
	}

	raw, err := googlemaps.PlaceDetails(ctx, placeID, sessionToken)
	if err != nil {
		if errors.Is(err, googlemaps.ErrNoResults) {
			return nil, errcustom.ErrNotFound
		}
		return nil, err
	}

	details := &models.PlaceDetails{
		Address: raw.Address,
		Lat:     raw.Lat,
		Lng:     raw.Lng,
	}

	if err := r.cache.Set(ctx, cacheKey, details, placeDetailsTTL); err != nil {
		slog.Error("maps: error al guardar el place details en el cache", "error", err)
	}

	return details, nil
}
