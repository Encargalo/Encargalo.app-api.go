package googlemaps

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// failingGroupTransport responde OK a todos los grupos menos al que contiene failOn.
type failingGroupTransport struct {
	failOn string
}

func (t *failingGroupTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	raw, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}

	var payload struct {
		IncludedPrimaryTypes []string `json:"includedPrimaryTypes"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}

	if slices.Contains(payload.IncludedPrimaryTypes, t.failOn) {
		return jsonResponse(http.StatusInternalServerError, `{"error":{"status":"INTERNAL","message":"boom"}}`), nil
	}

	body := fmt.Sprintf(`{"places":[{"id":%q,"displayName":{"text":"lugar"},"location":{"latitude":10.18,"longitude":-66.816}}]}`, payload.IncludedPrimaryTypes[0])
	return jsonResponse(http.StatusOK, body), nil
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}
}

func TestNearbyPopularPlaces_OneGroupDownIsPartial(t *testing.T) {
	original := httpClient
	httpClient = &http.Client{Transport: &failingGroupTransport{failOn: "train_station"}}
	t.Cleanup(func() { httpClient = original })

	places, err := NearbyPopularPlaces(context.Background(), 10.17809, -66.81615, 10000)

	require.ErrorIs(t, err, ErrPartialResults)
	require.Len(t, places, len(popularPlaceCategories)-1)
	require.Contains(t, err.Error(), "train_station")
	require.Contains(t, err.Error(), "INTERNAL boom")
}

func TestNearbyPopularPlaces_EveryGroupUpIsComplete(t *testing.T) {
	original := httpClient
	httpClient = &http.Client{Transport: &failingGroupTransport{failOn: "ningún-tipo"}}
	t.Cleanup(func() { httpClient = original })

	places, err := NearbyPopularPlaces(context.Background(), 10.17809, -66.81615, 10000)

	require.NoError(t, err)
	require.Len(t, places, len(popularPlaceCategories))
}
