package repo_test

import (
	"context"
	"encoding/json"
	"math"
	"testing"
	"time"

	"Encargalo.app-api.go/internal/maps/domain/models"
	"Encargalo.app-api.go/internal/maps/repo"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/stretchr/testify/require"
)

// stubCache sirve un valor fijo y registra el último Set: alcanza para los caminos que no llegan a Google.
type stubCache struct {
	value  string
	setKey string
	setTTL time.Duration
}

func (s *stubCache) Get(_ context.Context, _ string) (string, error) { return s.value, nil }

func (s *stubCache) Set(_ context.Context, key string, _ any, ttl time.Duration) error {
	s.setKey, s.setTTL = key, ttl
	return nil
}

func (s *stubCache) Clear(_ context.Context, _ string) error { return nil }

func (s *stubCache) ClearByPrefix(_ context.Context, _ string) error { return nil }

func TestPopularPlaces_CacheHitSkipsGoogle(t *testing.T) {
	cached, err := json.Marshal([]models.PopularPlace{{PlaceID: "abc", Name: "C.C. El Colonial"}})
	require.NoError(t, err)
	c := &stubCache{value: string(cached)}

	places, err := repo.NewMapsRepo(c).PopularPlaces(context.Background(), 10.17809, -66.81615, 10000)

	require.NoError(t, err)
	require.Len(t, places, 1)
	require.Equal(t, "abc", places[0].PlaceID)
	require.Empty(t, c.setKey)
}

func TestPopularPlaces_CachedEmptyListIsServedFromCache(t *testing.T) {
	c := &stubCache{value: "[]"}

	places, err := repo.NewMapsRepo(c).PopularPlaces(context.Background(), 10.17809, -66.81615, 10000)

	require.NoError(t, err)
	require.Empty(t, places)
	require.Empty(t, c.setKey)
}

func TestPopularPlaces_CoordinatesOutsideH3Domain(t *testing.T) {
	_, err := repo.NewMapsRepo(&stubCache{}).PopularPlaces(context.Background(), math.NaN(), -66.81615, 10000)

	require.ErrorIs(t, err, errcustom.ErrInvalidCoordinates)
}
