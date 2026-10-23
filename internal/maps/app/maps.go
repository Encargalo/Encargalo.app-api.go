package app

import (
	"context"
	"math"
	"sort"
	"strings"
	"unicode/utf8"

	"Encargalo.app-api.go/internal/maps/domain/dtos"
	"Encargalo.app-api.go/internal/maps/domain/models"
	"Encargalo.app-api.go/internal/maps/domain/ports"
	"Encargalo.app-api.go/internal/pkg/geo"
	"Encargalo.app-api.go/internal/shared/errcustom"
)

// minSearchQueryRunes evita que una búsqueda trivial (1-2 caracteres) le
// cueste dinero al backend en cada tecla — el móvil ya aplica el mismo
// mínimo por UX, pero el servidor no puede confiar en eso.
const minSearchQueryRunes = 3

const (
	// popularPlacesRadiusMeters no es configurable por el cliente: un radio fijo mantiene una sola entrada de cache por celda y alcanza para elegir un destino.
	popularPlacesRadiusMeters = 10000

	defaultPopularPlacesLimit = 20

	// El máximo lo rechaza el handler con un 400; acá se recorta igual para que la invariante valga para cualquier llamador del puerto.
	maxPopularPlacesLimit = 80
)

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

func (a *mapsApp) PopularPlaces(ctx context.Context, query dtos.PopularPlacesQuery) ([]models.PopularPlace, error) {
	if !validCoordinates(query.Latitude, query.Longitude) {
		return nil, errcustom.ErrInvalidCoordinates
	}

	limit := defaultPopularPlacesLimit
	if query.Limit > 0 {
		limit = min(query.Limit, maxPopularPlacesLimit)
	}

	places, err := a.repo.PopularPlaces(ctx, query.Latitude, query.Longitude, popularPlacesRadiusMeters)
	if err != nil {
		return nil, err
	}

	// El repo consulta el centro de la celda con un radio expandido, así que el radio real se aplica acá, contra la coordenada del usuario.
	inRadius := make([]models.PopularPlace, 0, len(places))
	for i := range places {
		km := geo.HaversineKm(query.Latitude, query.Longitude, places[i].Lat, places[i].Lng)
		if km*1000 > popularPlacesRadiusMeters {
			continue
		}
		places[i].DistanceKm = math.Round(km*100) / 100
		inRadius = append(inRadius, places[i])
	}
	places = inRadius

	sort.SliceStable(places, func(i, j int) bool {
		return places[i].DistanceKm < places[j].DistanceKm
	})

	if len(places) > limit {
		places = places[:limit]
	}

	return places, nil
}

func validCoordinates(lat, lng float64) bool {
	return lat >= -90 && lat <= 90 && lng >= -180 && lng <= 180
}
