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
	FormattedAddress string
	ShortAddress     string
}

// ReverseGeocode resolves a lat/lng pair into a human-readable address via
// the Google Geocoding API. Returns ErrNoResults if Google has no address
// for that coordinate. ctx is propagated so the call is canceled if the
// caller disconnects, and is bounded by httpClient's timeout either way.
func ReverseGeocode(ctx context.Context, lat, lng float64) (*ReverseGeocodeResult, error) {
	params := url.Values{}
	params.Set("latlng", fmt.Sprintf("%f,%f", lat, lng))
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
			FormattedAddress  string `json:"formatted_address"`
			AddressComponents []struct {
				LongName string `json:"long_name"`
			} `json:"address_components"`
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
	if len(data.Results) == 0 {
		return nil, ErrNoResults
	}

	result := data.Results[0]
	shortAddress := result.FormattedAddress
	if len(result.AddressComponents) > 0 {
		shortAddress = result.AddressComponents[0].LongName
	}

	return &ReverseGeocodeResult{
		FormattedAddress: result.FormattedAddress,
		ShortAddress:     shortAddress,
	}, nil
}
