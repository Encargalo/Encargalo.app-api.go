package googlemaps

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"

	"Encargalo.app-api.go/internal/shared/config"
)

// DistanceMeters returns the driving distance in meters between origin and destination.
func DistanceMeters(originLat, originLon, destLat, destLon float64) (float64, error) {
	params := url.Values{}
	params.Set("origins", fmt.Sprintf("%f,%f", originLat, originLon))
	params.Set("destinations", fmt.Sprintf("%f,%f", destLat, destLon))
	params.Set("mode", "driving")
	params.Set("key", config.Get().Google.ServerKey)

	fullURL := "https://maps.googleapis.com/maps/api/distancematrix/json?" + params.Encode()

	resp, err := httpClient.Get(fullURL) //nolint:noctx
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

// MatrixElement is one resolved destination from DistanceMatrix. Index is the
// position of the destination in the slice passed to DistanceMatrix, so
// callers can map it back to their own identifiers.
type MatrixElement struct {
	Index          int
	DistanceMeters float64
	DurationMin    int64
}

// DistanceMatrix returns driving distance/duration from one origin to up to
// 25 destinations in a single Google Distance Matrix call. Destinations
// Google can't resolve (element status != OK) are simply absent from the
// result, so callers must not assume len(result) == len(destinations).
func DistanceMatrix(originLat, originLon float64, destinations [][2]float64) ([]MatrixElement, error) {
	if len(destinations) == 0 {
		return nil, errors.New("no destinations provided")
	}
	if len(destinations) > 25 {
		return nil, fmt.Errorf("too many destinations: %d (max 25)", len(destinations))
	}

	destinationsRaw := make([]string, len(destinations))
	for i, d := range destinations {
		destinationsRaw[i] = fmt.Sprintf("%f,%f", d[0], d[1])
	}

	params := url.Values{}
	params.Set("origins", fmt.Sprintf("%f,%f", originLat, originLon))
	params.Set("destinations", strings.Join(destinationsRaw, "|"))
	params.Set("mode", "driving")
	params.Set("key", config.Get().Google.ServerKey)

	fullURL := "https://maps.googleapis.com/maps/api/distancematrix/json?" + params.Encode()

	resp, err := httpClient.Get(fullURL) //nolint:noctx
	if err != nil {
		return nil, fmt.Errorf("error calling google api: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %w", err)
	}
	if len(bodyBytes) == 0 {
		return nil, errors.New("empty response from google")
	}

	var data struct {
		Status string `json:"status"`
		Rows   []struct {
			Elements []struct {
				Distance struct {
					Value float64 `json:"value"`
				} `json:"distance"`
				Duration struct {
					Value int64 `json:"value"`
				} `json:"duration"`
				Status string `json:"status"`
			} `json:"elements"`
		} `json:"rows"`
	}

	if err := json.Unmarshal(bodyBytes, &data); err != nil {
		return nil, fmt.Errorf("decode json error: %w; body=%s", err, string(bodyBytes))
	}
	if data.Status != "OK" {
		return nil, fmt.Errorf("google api status=%s", data.Status)
	}
	if len(data.Rows) == 0 {
		return nil, errors.New("invalid google matrix response")
	}

	elements := data.Rows[0].Elements
	result := make([]MatrixElement, 0, len(elements))
	for i, elem := range elements {
		if elem.Status != "OK" {
			continue
		}
		result = append(result, MatrixElement{
			Index:          i,
			DistanceMeters: elem.Distance.Value,
			DurationMin:    elem.Duration.Value / 60,
		})
	}

	return result, nil
}
