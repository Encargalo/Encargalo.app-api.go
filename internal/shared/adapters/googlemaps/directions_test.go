package googlemaps

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseDirectionsResponse_OK(t *testing.T) {
	body := []byte(`{"status":"OK","routes":[{"overview_polyline":{"points":"abc"},"legs":[{"distance":{"value":1234},"duration":{"value":330}}]}]}`)

	route, err := parseDirectionsResponse(body)

	require.NoError(t, err)
	require.Equal(t, "abc", route.Polyline)
	require.Equal(t, int64(1234), route.DistanceMeters)
	require.Equal(t, int64(5), route.DurationMin)
}

func TestParseDirectionsResponse_NoRouteIsNoResults(t *testing.T) {
	for _, body := range []string{
		`{"status":"ZERO_RESULTS","routes":[]}`,
		`{"status":"NOT_FOUND","routes":[]}`,
		`{"status":"OK","routes":[]}`,
	} {
		_, err := parseDirectionsResponse([]byte(body))
		require.ErrorIs(t, err, ErrNoResults, body)
	}
}

func TestParseDirectionsResponse_ErrorStatus(t *testing.T) {
	_, err := parseDirectionsResponse([]byte(`{"status":"REQUEST_DENIED","routes":[]}`))

	require.Error(t, err)
	require.NotErrorIs(t, err, ErrNoResults)
	require.Contains(t, err.Error(), "REQUEST_DENIED")
}

func TestParseDirectionsResponse_InvalidJSON(t *testing.T) {
	_, err := parseDirectionsResponse([]byte(`<html>`))

	require.Error(t, err)
	require.NotErrorIs(t, err, ErrNoResults)
}
