package googlemaps

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"Encargalo.app-api.go/internal/shared/config"
)

// DistanceMeters returns the driving distance in meters between origin and destination.
func DistanceMeters(originLat, originLon, destLat, destLon float64) (float64, error) {
	params := url.Values{}
	params.Set("origins", fmt.Sprintf("%f,%f", originLat, originLon))
	params.Set("destinations", fmt.Sprintf("%f,%f", destLat, destLon))
	params.Set("mode", "driving")
	params.Set("key", config.Get().Google.Matrix)

	fullURL := "https://maps.googleapis.com/maps/api/distancematrix/json?" + params.Encode()

	resp, err := http.Get(fullURL) //nolint:noctx
	if err != nil {
		return 0, fmt.Errorf("google maps request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("reading google maps response: %w", err)
	}

	var data struct {
		Status string `json:"status"`
		Rows   []struct {
			Elements []struct {
				Distance struct {
					Value float64 `json:"value"`
				} `json:"distance"`
				Status string `json:"status"`
			} `json:"elements"`
		} `json:"rows"`
	}

	if err := json.Unmarshal(bodyBytes, &data); err != nil {
		return 0, fmt.Errorf("decoding google maps response: %w", err)
	}

	if data.Status != "OK" {
		return 0, fmt.Errorf("google maps api status: %s", data.Status)
	}

	if len(data.Rows) == 0 || len(data.Rows[0].Elements) == 0 {
		return 0, errors.New("empty google maps response")
	}

	elem := data.Rows[0].Elements[0]
	if elem.Status != "OK" {
		return 0, fmt.Errorf("google maps element status: %s", elem.Status)
	}

	return elem.Distance.Value, nil
}
