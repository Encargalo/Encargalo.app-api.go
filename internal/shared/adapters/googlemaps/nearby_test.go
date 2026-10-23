package googlemaps

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseNearbyResponse_MapsEveryField(t *testing.T) {
	body := []byte(`{"places":[{"id":"abc","displayName":{"text":"C.C. El Colonial"},"primaryTypeDisplayName":{"text":"Centro comercial"},"shortFormattedAddress":"Av. Intercomunal, Guarenas","location":{"latitude":10.4712,"longitude":-66.6103}}]}`)

	places, err := parseNearbyResponse(body)

	require.NoError(t, err)
	require.Equal(t, []NearbyPlace{{
		PlaceID:  "abc",
		Name:     "C.C. El Colonial",
		Category: "Centro comercial",
		Address:  "Av. Intercomunal, Guarenas",
		Lat:      10.4712,
		Lng:      -66.6103,
	}}, places)
}

func TestParseNearbyResponse_NoPlacesIsNotAnError(t *testing.T) {
	places, err := parseNearbyResponse([]byte(`{}`))

	require.NoError(t, err)
	require.Empty(t, places)
}

func TestParseNearbyResponse_InvalidJSON(t *testing.T) {
	_, err := parseNearbyResponse([]byte(`<html>`))

	require.Error(t, err)
	require.NotErrorIs(t, err, ErrNoResults)
}

func TestMergeNearbyResults_DedupesByPlaceID(t *testing.T) {
	results := [][]NearbyPlace{
		{{PlaceID: "a"}, {PlaceID: "b"}},
		{{PlaceID: "b"}, {PlaceID: "c"}},
	}

	merged, err := mergeNearbyResults(results, []error{nil, nil})

	require.NoError(t, err)
	require.Len(t, merged, 3)
	require.Equal(t, []NearbyPlace{{PlaceID: "a"}, {PlaceID: "b"}, {PlaceID: "c"}}, merged)
}

func TestMergeNearbyResults_KeepsResultsOfTheCategoriesThatWorked(t *testing.T) {
	results := [][]NearbyPlace{
		nil,
		{{PlaceID: "b"}},
	}
	boom := errors.New("boom")

	merged, err := mergeNearbyResults(results, []error{boom, nil})

	require.Equal(t, []NearbyPlace{{PlaceID: "b"}}, merged)
	require.ErrorIs(t, err, ErrPartialResults)
	require.ErrorIs(t, err, boom)
}

func TestMergeNearbyResults_PartialWithoutPlacesIsStillPartial(t *testing.T) {
	merged, err := mergeNearbyResults([][]NearbyPlace{nil, {}}, []error{errors.New("boom"), nil})

	require.Empty(t, merged)
	require.ErrorIs(t, err, ErrPartialResults)
	require.NotErrorIs(t, err, ErrNoResults)
}

func TestMergeNearbyResults_EveryCategoryFailed(t *testing.T) {
	first := errors.New("boom")
	second := errors.New("otra")

	_, err := mergeNearbyResults([][]NearbyPlace{nil, nil}, []error{first, second})

	require.ErrorIs(t, err, first)
	require.ErrorIs(t, err, second)
	require.NotErrorIs(t, err, ErrPartialResults)
	require.NotErrorIs(t, err, ErrNoResults)
}

func TestMergeNearbyResults_NoPlacesIsNoResults(t *testing.T) {
	_, err := mergeNearbyResults([][]NearbyPlace{{}, {}}, []error{nil, nil})

	require.ErrorIs(t, err, ErrNoResults)
}

func TestNearbyErrorDetail_ReadsGoogleError(t *testing.T) {
	detail := nearbyErrorDetail([]byte(`{"error":{"status":"INVALID_ARGUMENT","message":"includedPrimaryTypes contains an invalid type"}}`))

	require.Equal(t, "INVALID_ARGUMENT includedPrimaryTypes contains an invalid type", detail)
}

func TestNearbyErrorDetail_TruncatesANonJSONBody(t *testing.T) {
	detail := nearbyErrorDetail([]byte("<html>" + strings.Repeat("x", 500)))

	require.Len(t, detail, nearbyErrorBodyLimit)
}
