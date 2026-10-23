package repo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"time"

	"Encargalo.app-api.go/internal/maps/domain/models"
	"Encargalo.app-api.go/internal/maps/domain/ports"
	"Encargalo.app-api.go/internal/pkg/cache"
	"Encargalo.app-api.go/internal/pkg/geo"
	"Encargalo.app-api.go/internal/shared/adapters/googlemaps"
	"Encargalo.app-api.go/internal/shared/config"
	"Encargalo.app-api.go/internal/shared/errcustom"
)

const (
	// v3: payload now includes every Google result; bumping the prefix
	// discards cached responses that only had results[0].
	reverseGeocodePrefix = "maps:reverse_geocode:v3:"
	reverseGeocodeTTL    = 15 * time.Minute

	placeDetailsPrefix = "maps:place_details:v2:"
	placeDetailsTTL    = 24 * time.Hour

	// v2: la key ya no lleva el radio (lo fija el app) y el payload se arma con el radio expandido por el circunradio de la celda.
	popularPlacesPrefix = "maps:popular_places:v2:"
	popularPlacesTTL    = 7 * 24 * time.Hour

	// Un resultado parcial se cachea corto: sirve para responder ya, pero no para fijar por una semana una lista a la que le falta un grupo de categorías.
	popularPlacesPartialTTL = 15 * time.Minute

	// Una zona sin lugares populares se cachea corto: evita repetir 4 llamadas facturadas por request sin fijar por una semana lo que puede ser un vacío transitorio.
	popularPlacesEmptyTTL = time.Hour

	// Granularidad del cache de lugares populares (celdas de ~5.16 km²); propia del módulo, no la resolución de los heatmaps de analítica.
	popularPlacesCellResolution = 7

	// Circunradio de una celda de esa resolución: el usuario puede estar a ~1.22 km del centro que se le consulta a Google.
	popularPlacesCellRadiusMeters = 1300
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
		FormattedAddress:  raw.FormattedAddress,
		ShortAddress:      raw.ShortAddress,
		Parts:             toDomainParts(raw.Parts),
		AddressComponents: toDomainComponents(raw.AddressComponents),
		Results:           toDomainCandidates(raw.Results),
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
		Address:           raw.Address,
		Lat:               raw.Lat,
		Lng:               raw.Lng,
		Parts:             toDomainParts(raw.Parts),
		AddressComponents: toDomainComponents(raw.AddressComponents),
	}

	if err := r.cache.Set(ctx, cacheKey, details, placeDetailsTTL); err != nil {
		slog.Error("maps: error al guardar el place details en el cache", "error", err)
	}

	return details, nil
}

// Directions is not cached: the rider-to-pickup origin moves with every refresh.
// Outside production it returns a straight line to avoid spending Google quota.
func (r *mapsRepo) Directions(ctx context.Context, originLat, originLng, destLat, destLng float64) (*models.Route, error) {
	if !config.Get().IsProduction() {
		return straightRoute(originLat, originLng, destLat, destLng), nil
	}

	raw, err := googlemaps.Directions(ctx, originLat, originLng, destLat, destLng)
	if errors.Is(err, googlemaps.ErrNoResults) {
		return nil, errcustom.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errcustom.ErrMapsUnavailable, err)
	}

	return &models.Route{
		Polyline:       raw.Polyline,
		DistanceMeters: raw.DistanceMeters,
		DurationMin:    raw.DurationMin,
	}, nil
}

func (r *mapsRepo) PopularPlaces(ctx context.Context, lat, lng float64, radiusMeters int) ([]models.PopularPlace, error) {
	cell, err := geo.LatLngToCell(lat, lng, popularPlacesCellResolution)
	if err != nil {
		return nil, errcustom.ErrInvalidCoordinates
	}

	// La key no incluye el radio porque el app lo fija: una sola entrada por celda, compartida por todos los usuarios de la zona.
	cacheKey := popularPlacesPrefix + cell

	cached, _ := r.cache.Get(ctx, cacheKey)
	if cached != "" {
		var places []models.PopularPlace
		if err := json.Unmarshal([]byte(cached), &places); err == nil {
			return places, nil
		}
		slog.Error("maps: error al hacer unmarshal de los lugares populares cacheados", "key", cacheKey)
	}

	centerLat, centerLng, err := geo.CellToLatLng(cell)
	if err != nil {
		return nil, errcustom.ErrInvalidCoordinates
	}

	// Se pide el radio expandido por el circunradio de la celda para no perder lugares cercanos al usuario pero lejanos al centro consultado.
	raw, err := googlemaps.NearbyPopularPlaces(ctx, centerLat, centerLng, radiusMeters+popularPlacesCellRadiusMeters)
	if errors.Is(err, googlemaps.ErrNoResults) {
		empty := []models.PopularPlace{}
		if err := r.cache.Set(ctx, cacheKey, empty, popularPlacesEmptyTTL); err != nil {
			slog.Error("maps: error al guardar los lugares populares vacíos en el cache", "error", err)
		}
		return empty, nil
	}

	partial := errors.Is(err, googlemaps.ErrPartialResults)
	if err != nil && !partial {
		return nil, fmt.Errorf("%w: %v", errcustom.ErrMapsUnavailable, err)
	}

	places := make([]models.PopularPlace, 0, len(raw))
	for _, p := range raw {
		places = append(places, models.PopularPlace{
			PlaceID:  p.PlaceID,
			Name:     p.Name,
			Category: p.Category,
			Address:  p.Address,
			Lat:      p.Lat,
			Lng:      p.Lng,
		})
	}

	ttl := popularPlacesTTL
	if partial {
		slog.Warn("maps: lugares populares incompletos, falló al menos un grupo de categorías", "cell", cell, "error", err)
		ttl = popularPlacesPartialTTL
	}

	if err := r.cache.Set(ctx, cacheKey, places, ttl); err != nil {
		slog.Error("maps: error al guardar los lugares populares en el cache", "error", err)
	}

	return places, nil
}

func straightRoute(originLat, originLng, destLat, destLng float64) *models.Route {
	return &models.Route{
		Polyline:       geo.EncodePolyline([][2]float64{{originLat, originLng}, {destLat, destLng}}),
		DistanceMeters: int64(math.Round(geo.HaversineKm(originLat, originLng, destLat, destLng) * 1000)),
	}
}

func toDomainParts(src googlemaps.AddressParts) models.AddressParts {
	return models.AddressParts{
		StreetNumber:             src.StreetNumber,
		Route:                    src.Route,
		Neighborhood:             src.Neighborhood,
		Sublocality:              src.Sublocality,
		Locality:                 src.Locality,
		AdministrativeAreaLevel3: src.AdministrativeAreaLevel3,
		AdministrativeAreaLevel2: src.AdministrativeAreaLevel2,
		AdministrativeAreaLevel1: src.AdministrativeAreaLevel1,
		Country:                  src.Country,
		CountryCode:              src.CountryCode,
		PostalCode:               src.PostalCode,
		Premise:                  src.Premise,
		Subpremise:               src.Subpremise,
	}
}

func toDomainComponents(src []googlemaps.AddressComponent) []models.AddressComponent {
	components := make([]models.AddressComponent, 0, len(src))
	for _, c := range src {
		components = append(components, models.AddressComponent{
			LongName:  c.LongName,
			ShortName: c.ShortName,
			Types:     c.Types,
		})
	}
	return components
}

func toDomainCandidates(src []googlemaps.GeocodeCandidate) []models.GeocodeCandidate {
	candidates := make([]models.GeocodeCandidate, 0, len(src))
	for _, c := range src {
		candidates = append(candidates, models.GeocodeCandidate{
			FormattedAddress:  c.FormattedAddress,
			PlaceID:           c.PlaceID,
			Types:             c.Types,
			Lat:               c.Lat,
			Lng:               c.Lng,
			LocationType:      c.LocationType,
			Parts:             toDomainParts(c.Parts),
			AddressComponents: toDomainComponents(c.AddressComponents),
		})
	}
	return candidates
}
