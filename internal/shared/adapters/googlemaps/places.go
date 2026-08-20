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

// PlacePrediction is one autocomplete suggestion from the Places API.
type PlacePrediction struct {
	PlaceID       string
	MainText      string
	SecondaryText string
}

// PlaceDetailsResult is the resolved address/coordinates for a place_id.
type PlaceDetailsResult struct {
	Address           string
	Lat               float64
	Lng               float64
	Parts             AddressParts
	AddressComponents []AddressComponent
}

// AutocompletePlaces returns place predictions for query, optionally biased
// towards biasLat/biasLng. sessionToken groups this call with the eventual
// PlaceDetails call under one billing session, per Google's Session Token
// guidance. ctx is propagated so the call is canceled if the caller
// disconnects, and is bounded by httpClient's timeout either way.
func AutocompletePlaces(ctx context.Context, query string, biasLat, biasLng *float64, sessionToken string) ([]PlacePrediction, error) {
	params := url.Values{}
	params.Set("input", query)
	params.Set("sessiontoken", sessionToken)
	params.Set("key", config.Get().Google.ServerKey)
	if biasLat != nil && biasLng != nil {
		params.Set("location", fmt.Sprintf("%f,%f", *biasLat, *biasLng))
		params.Set("radius", "20000")
	}

	fullURL := "https://maps.googleapis.com/maps/api/place/autocomplete/json?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("building google places autocomplete request: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("google places autocomplete request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading google places autocomplete response: %w", err)
	}

	var data struct {
		Status      string `json:"status"`
		Predictions []struct {
			PlaceID              string `json:"place_id"`
			StructuredFormatting struct {
				MainText      string `json:"main_text"`
				SecondaryText string `json:"secondary_text"`
			} `json:"structured_formatting"`
		} `json:"predictions"`
	}

	if err := json.Unmarshal(bodyBytes, &data); err != nil {
		return nil, fmt.Errorf("decoding google places autocomplete response: %w", err)
	}

	if data.Status != "OK" && data.Status != "ZERO_RESULTS" {
		return nil, fmt.Errorf("google places autocomplete api status: %s", data.Status)
	}

	predictions := make([]PlacePrediction, 0, len(data.Predictions))
	for _, p := range data.Predictions {
		predictions = append(predictions, PlacePrediction{
			PlaceID:       p.PlaceID,
			MainText:      p.StructuredFormatting.MainText,
			SecondaryText: p.StructuredFormatting.SecondaryText,
		})
	}

	return predictions, nil
}

// PlaceDetails resolves the formatted address and coordinates for placeID.
// sessionToken must match the one used in the preceding AutocompletePlaces
// call to close out the billing session. Returns ErrNoResults if Google has
// nothing for that place_id.
func PlaceDetails(ctx context.Context, placeID, sessionToken string) (*PlaceDetailsResult, error) {
	params := url.Values{}
	params.Set("place_id", placeID)
	params.Set("fields", "geometry,formatted_address,address_component")
	params.Set("sessiontoken", sessionToken)
	params.Set("key", config.Get().Google.ServerKey)

	fullURL := "https://maps.googleapis.com/maps/api/place/details/json?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("building google place details request: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("google place details request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading google place details response: %w", err)
	}

	var data struct {
		Status string `json:"status"`
		Result struct {
			FormattedAddress  string                   `json:"formatted_address"`
			AddressComponents []googleAddressComponent `json:"address_components"`
			Geometry          struct {
				Location struct {
					Lat float64 `json:"lat"`
					Lng float64 `json:"lng"`
				} `json:"location"`
			} `json:"geometry"`
		} `json:"result"`
	}

	if err := json.Unmarshal(bodyBytes, &data); err != nil {
		return nil, fmt.Errorf("decoding google place details response: %w", err)
	}

	if data.Status == "ZERO_RESULTS" || data.Status == "NOT_FOUND" {
		return nil, ErrNoResults
	}
	if data.Status != "OK" {
		return nil, fmt.Errorf("google place details api status: %s", data.Status)
	}
	if data.Result.FormattedAddress == "" {
		return nil, ErrNoResults
	}

	components, parts := parseAddressComponents(data.Result.AddressComponents)

	return &PlaceDetailsResult{
		Address:           data.Result.FormattedAddress,
		Lat:               data.Result.Geometry.Location.Lat,
		Lng:               data.Result.Geometry.Location.Lng,
		Parts:             parts,
		AddressComponents: components,
	}, nil
}
