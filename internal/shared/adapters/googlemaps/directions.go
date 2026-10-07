package googlemaps

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"Encargalo.app-api.go/internal/shared/config"
	"Encargalo.app-api.go/internal/shared/reqctx"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
)

// DirectionsRoute es la ruta en auto entre dos puntos según Google Directions.
type DirectionsRoute struct {
	Polyline       string
	DistanceMeters int64
	DurationMin    int64
}

// Directions devuelve la ruta en auto entre origen y destino. ErrNoResults si
// Google no encuentra ruta (ZERO_RESULTS o NOT_FOUND).
func Directions(ctx context.Context, originLat, originLng, destLat, destLng float64) (*DirectionsRoute, error) {
	ctx, span := otel.Tracer(reqctx.ServiceName).Start(ctx, "googlemaps.Directions")
	defer span.End()

	params := url.Values{}
	params.Set("origin", fmt.Sprintf("%f,%f", originLat, originLng))
	params.Set("destination", fmt.Sprintf("%f,%f", destLat, destLng))
	params.Set("mode", "driving")
	params.Set("key", config.Get().Google.ServerKey)

	fullURL := "https://maps.googleapis.com/maps/api/directions/json?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("building google directions request: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("google directions request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("reading google directions response: %w", err)
	}

	route, err := parseDirectionsResponse(bodyBytes)
	if err != nil && !errors.Is(err, ErrNoResults) {
		span.SetStatus(codes.Error, err.Error())
	}
	return route, err
}

func parseDirectionsResponse(body []byte) (*DirectionsRoute, error) {
	var data struct {
		Status string `json:"status"`
		Routes []struct {
			OverviewPolyline struct {
				Points string `json:"points"`
			} `json:"overview_polyline"`
			Legs []struct {
				Distance struct {
					Value int64 `json:"value"`
				} `json:"distance"`
				Duration struct {
					Value int64 `json:"value"`
				} `json:"duration"`
			} `json:"legs"`
		} `json:"routes"`
	}

	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("decoding google directions response: %w", err)
	}

	switch data.Status {
	case "OK":
	case "ZERO_RESULTS", "NOT_FOUND":
		return nil, ErrNoResults
	default:
		return nil, fmt.Errorf("google directions api status: %s", data.Status)
	}

	if len(data.Routes) == 0 || data.Routes[0].OverviewPolyline.Points == "" {
		return nil, ErrNoResults
	}

	route := data.Routes[0]
	result := &DirectionsRoute{Polyline: route.OverviewPolyline.Points}
	if len(route.Legs) > 0 {
		result.DistanceMeters = route.Legs[0].Distance.Value
		result.DurationMin = route.Legs[0].Duration.Value / 60
	}
	return result, nil
}
