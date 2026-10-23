package googlemaps

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"Encargalo.app-api.go/internal/shared/config"
)

// ReverseGeocodeResult is the resolved address for a lat/lng pair.
type ReverseGeocodeResult struct {
	FormattedAddress  string
	ShortAddress      string
	Parts             AddressParts
	AddressComponents []AddressComponent
	Results           []GeocodeCandidate
}

// GeocodeCandidate is one entry of Google's results array, kept whole because
// the first one often lacks street_number while a later one has it.
type GeocodeCandidate struct {
	FormattedAddress  string
	PlaceID           string
	Types             []string
	Lat               float64
	Lng               float64
	LocationType      string
	Parts             AddressParts
	AddressComponents []AddressComponent
}

// ReverseGeocode resolves a lat/lng pair into a human-readable address via
// the Google Geocoding API. Returns ErrNoResults if Google has no address
// for that coordinate. ctx is propagated so the call is canceled if the
// caller disconnects, and is bounded by httpClient's timeout either way.
func ReverseGeocode(ctx context.Context, lat, lng float64) (*ReverseGeocodeResult, error) {
	params := url.Values{}
	params.Set("latlng", fmt.Sprintf("%f,%f", lat, lng))
	params.Set("language", languageCode())
	params.Set("key", config.Get().Google.ServerKey)

	fullURL := "https://maps.googleapis.com/maps/api/geocode/json?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("building google geocoding request: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("google geocoding request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading google geocoding response: %w", err)
	}

	var data struct {
		Status  string `json:"status"`
		Results []struct {
			FormattedAddress  string                   `json:"formatted_address"`
			PlaceID           string                   `json:"place_id"`
			Types             []string                 `json:"types"`
			AddressComponents []googleAddressComponent `json:"address_components"`
			Geometry          struct {
				Location struct {
					Lat float64 `json:"lat"`
					Lng float64 `json:"lng"`
				} `json:"location"`
				LocationType string `json:"location_type"`
			} `json:"geometry"`
		} `json:"results"`
	}

	if err := json.Unmarshal(bodyBytes, &data); err != nil {
		return nil, fmt.Errorf("decoding google geocoding response: %w", err)
	}

	if data.Status == "ZERO_RESULTS" {
		return nil, ErrNoResults
	}
	if data.Status != "OK" {
		return nil, fmt.Errorf("google geocoding api status: %s", data.Status)
	}

	// Reverse geocoding no acepta components=country, así que los candidatos de fuera del área de servicio se descartan acá.
	candidates := make([]GeocodeCandidate, 0, len(data.Results))
	for _, r := range data.Results {
		components, parts := parseAddressComponents(r.AddressComponents)
		if !matchesRegion(parts.CountryCode) {
			continue
		}
		candidates = append(candidates, GeocodeCandidate{
			FormattedAddress:  r.FormattedAddress,
			PlaceID:           r.PlaceID,
			Types:             r.Types,
			Lat:               r.Geometry.Location.Lat,
			Lng:               r.Geometry.Location.Lng,
			LocationType:      r.Geometry.LocationType,
			Parts:             parts,
			AddressComponents: components,
		})
	}

	if len(candidates) == 0 {
		return nil, ErrNoResults
	}

	first := candidates[0]
	shortAddress := first.FormattedAddress
	if len(first.AddressComponents) > 0 {
		shortAddress = first.AddressComponents[0].LongName
	}

	return &ReverseGeocodeResult{
		FormattedAddress:  first.FormattedAddress,
		ShortAddress:      shortAddress,
		Parts:             first.Parts,
		AddressComponents: first.AddressComponents,
		Results:           candidates,
	}, nil
}
