package googlemaps

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"Encargalo.app-api.go/internal/shared/config"
)

const (
	nearbySearchURL       = "https://places.googleapis.com/v1/places:searchNearby"
	nearbySearchFieldMask = "places.id,places.displayName,places.location,places.primaryTypeDisplayName,places.shortFormattedAddress"
	nearbyMaxResultCount  = 20

	// maxNearbySearchRadiusMeters es el tope que acepta Nearby Search; pedir más es un 400 de Google.
	maxNearbySearchRadiusMeters = 50000

	// nearbyErrorBodyLimit acota el cuerpo crudo que se adjunta al error cuando Google no responde JSON.
	nearbyErrorBodyLimit = 200
)

var popularPlaceCategories = [][]string{
	{"train_station", "bus_station", "transit_station", "subway_station", "airport"},
	{"shopping_mall", "supermarket", "grocery_store", "department_store", "wholesaler"},
	{"hospital", "pharmacy", "doctor"},
	{"bank", "park", "tourist_attraction", "stadium", "university", "church"},
}

// MaxNearbyPopularPlaces es el techo real de lugares distintos que puede devolver NearbyPopularPlaces.
var MaxNearbyPopularPlaces = len(popularPlaceCategories) * nearbyMaxResultCount

type NearbyPlace struct {
	PlaceID  string
	Name     string
	Category string
	Address  string
	Lat      float64
	Lng      float64
}

func NearbyPopularPlaces(ctx context.Context, lat, lng float64, radiusMeters int) ([]NearbyPlace, error) {
	var (
		wg      sync.WaitGroup
		results = make([][]NearbyPlace, len(popularPlaceCategories))
		errs    = make([]error, len(popularPlaceCategories))
	)

	// Cada goroutine escribe en su propio índice de slices ya dimensionados: no hay escritura compartida que sincronizar.
	for i, types := range popularPlaceCategories {
		wg.Add(1)
		go func(idx int, included []string) {
			defer wg.Done()
			places, err := searchNearby(ctx, lat, lng, radiusMeters, included, "POPULARITY", nearbyMaxResultCount)
			if err != nil {
				err = fmt.Errorf("categorías %v: %w", included, err)
			}
			results[idx], errs[idx] = places, err
		}(i, types)
	}
	wg.Wait()

	return mergeNearbyResults(results, errs)
}

// NearestPlace devuelve el lugar con nombre más cercano a una coordenada, sin
// restringir por categoría — a diferencia de NearbyPopularPlaces, acá no
// importa qué tan popular es, solo qué tan cerca está. Devuelve ErrNoResults
// si no hay ningún lugar con nombre dentro de radiusMeters.
func NearestPlace(ctx context.Context, lat, lng float64, radiusMeters int) (*NearbyPlace, error) {
	places, err := searchNearby(ctx, lat, lng, radiusMeters, nil, "DISTANCE", 1)
	if err != nil {
		return nil, err
	}
	if len(places) == 0 {
		return nil, ErrNoResults
	}
	return &places[0], nil
}

// mergeNearbyResults une los resultados de cada grupo de categorías descartando los place_id repetidos.
func mergeNearbyResults(results [][]NearbyPlace, errs []error) ([]NearbyPlace, error) {
	merged := make([]NearbyPlace, 0, len(results)*nearbyMaxResultCount)
	seen := make(map[string]struct{}, cap(merged))
	failed := 0

	for i, places := range results {
		if errs[i] != nil {
			failed++
			continue
		}
		for _, p := range places {
			if _, dup := seen[p.PlaceID]; dup {
				continue
			}
			seen[p.PlaceID] = struct{}{}
			merged = append(merged, p)
		}
	}

	if failed > 0 && failed == len(results) {
		return nil, fmt.Errorf("google nearby search failed for every category: %w", errors.Join(errs...))
	}

	// Resultado parcial: se devuelven los lugares que sí llegaron, pero el llamador tiene que poder enterarse para no tratarlos como una lista completa.
	if failed > 0 {
		return merged, fmt.Errorf("%w: %w", ErrPartialResults, errors.Join(errs...))
	}

	if len(merged) == 0 {
		return nil, ErrNoResults
	}

	return merged, nil
}

func searchNearby(ctx context.Context, lat, lng float64, radiusMeters int, includedPrimaryTypes []string, rankPreference string, maxResultCount int) ([]NearbyPlace, error) {
	payload := map[string]any{
		"locationRestriction": map[string]any{
			"circle": map[string]any{
				"center": map[string]any{"latitude": lat, "longitude": lng},
				"radius": min(radiusMeters, maxNearbySearchRadiusMeters),
			},
		},
		"rankPreference": rankPreference,
		"maxResultCount": maxResultCount,
		"languageCode":   languageCode(),
		"regionCode":     regionCode(),
	}
	// Omitido en vez de mandado vacío: un array vacío en includedPrimaryTypes lo rechaza Google con 400, en vez de significar "cualquier categoría".
	if len(includedPrimaryTypes) > 0 {
		payload["includedPrimaryTypes"] = includedPrimaryTypes
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("building google nearby search body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, nearbySearchURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("building google nearby search request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Goog-Api-Key", config.Get().Google.ServerKey)
	req.Header.Set("X-Goog-FieldMask", nearbySearchFieldMask)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("google nearby search request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading google nearby search response: %w", err)
	}

	// El status se revisa antes de parsear: un 5xx de un proxy no trae JSON y el error debe decir el código, no "decoding".
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google nearby search api status %d: %s", resp.StatusCode, nearbyErrorDetail(bodyBytes))
	}

	return parseNearbyResponse(bodyBytes)
}

func parseNearbyResponse(body []byte) ([]NearbyPlace, error) {
	var data struct {
		Places []struct {
			ID          string `json:"id"`
			DisplayName struct {
				Text string `json:"text"`
			} `json:"displayName"`
			PrimaryTypeDisplayName struct {
				Text string `json:"text"`
			} `json:"primaryTypeDisplayName"`
			ShortFormattedAddress string `json:"shortFormattedAddress"`
			Location              struct {
				Latitude  float64 `json:"latitude"`
				Longitude float64 `json:"longitude"`
			} `json:"location"`
		} `json:"places"`
	}

	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("decoding google nearby search response: %w", err)
	}

	places := make([]NearbyPlace, 0, len(data.Places))
	for _, p := range data.Places {
		places = append(places, NearbyPlace{
			PlaceID:  p.ID,
			Name:     p.DisplayName.Text,
			Category: p.PrimaryTypeDisplayName.Text,
			Address:  p.ShortFormattedAddress,
			Lat:      p.Location.Latitude,
			Lng:      p.Location.Longitude,
		})
	}

	return places, nil
}

// nearbyErrorDetail extrae status/message del error de Google, o un recorte del cuerpo si no es el JSON esperado.
func nearbyErrorDetail(body []byte) string {
	var data struct {
		Error struct {
			Status  string `json:"status"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &data); err != nil || data.Error.Status == "" {
		return strings.TrimSpace(string(body[:min(len(body), nearbyErrorBodyLimit)]))
	}

	return strings.TrimSpace(data.Error.Status + " " + data.Error.Message)
}
