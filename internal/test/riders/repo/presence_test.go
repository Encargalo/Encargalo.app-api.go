package repo_test

import (
	"context"
	"testing"

	"Encargalo.app-api.go/internal/riders/domain/models"
	"Encargalo.app-api.go/internal/riders/repo"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func newPresenceRepoForTest(t *testing.T) *redis.Client {
	t.Helper()
	client := redis.NewClient(&redis.Options{Addr: miniredis.RunT(t).Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func TestRiderPresence_IsPaused_DefaultsFalse(t *testing.T) {
	store := repo.NewRiderPresenceRepo(newPresenceRepoForTest(t))
	ctx := context.Background()

	paused, err := store.IsPaused(ctx, uuid.New())

	require.NoError(t, err)
	require.False(t, paused)
}

func TestRiderPresence_SetPaused_TrueThenFalse(t *testing.T) {
	store := repo.NewRiderPresenceRepo(newPresenceRepoForTest(t))
	ctx := context.Background()
	riderID := uuid.New()

	require.NoError(t, store.SetPaused(ctx, riderID, true))
	paused, err := store.IsPaused(ctx, riderID)
	require.NoError(t, err)
	require.True(t, paused)

	require.NoError(t, store.SetPaused(ctx, riderID, false))
	paused, err = store.IsPaused(ctx, riderID)
	require.NoError(t, err)
	require.False(t, paused)
}

// La pausa vive en una clave propia: Remove (el borrado completo que dispara
// un Unregister) no debe quitarle la pausa a un rider que sigue conectado por
// otro socket, o que se reconecta por una caída de red sin pedirlo.
func TestRiderPresence_Remove_DoesNotClearPaused(t *testing.T) {
	store := repo.NewRiderPresenceRepo(newPresenceRepoForTest(t))
	ctx := context.Background()
	riderID := uuid.New()

	require.NoError(t, store.SetPaused(ctx, riderID, true))
	require.NoError(t, store.Remove(ctx, riderID))

	paused, err := store.IsPaused(ctx, riderID)
	require.NoError(t, err)
	require.True(t, paused)
}

// SaveLocation hace un HSET parcial sobre el hash de presencia: no debe tocar
// la clave de pausa, que vive aparte.
func TestRiderPresence_SaveLocation_DoesNotClearPaused(t *testing.T) {
	store := repo.NewRiderPresenceRepo(newPresenceRepoForTest(t))
	ctx := context.Background()
	riderID := uuid.New()

	require.NoError(t, store.SetPaused(ctx, riderID, true))
	require.NoError(t, store.SaveLocation(ctx, models.RiderPresence{
		RiderID: riderID,
		Location: models.RiderLocation{
			Latitude:  10.48,
			Longitude: -66.90,
		},
		Mode: models.RiderModeTrips,
	}))

	paused, err := store.IsPaused(ctx, riderID)
	require.NoError(t, err)
	require.True(t, paused)
}
