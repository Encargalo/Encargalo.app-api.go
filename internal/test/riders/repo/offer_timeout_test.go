package repo_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"Encargalo.app-api.go/internal/riders/repo"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func newOfferTimeoutStoreForTest(t *testing.T) *redis.Client {
	t.Helper()
	client := redis.NewClient(&redis.Options{Addr: miniredis.RunT(t).Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return client
}

// TestTripOfferTimeout_PopDue_OnlyReturnsExpired es el corazón de #359: un
// vencimiento futuro no debe salir en PopDue, solo los ya cumplidos.
func TestTripOfferTimeout_PopDue_OnlyReturnsExpired(t *testing.T) {
	store := repo.NewTripOfferTimeoutRepo(newOfferTimeoutStoreForTest(t))
	ctx := context.Background()

	due := uuid.New()
	future := uuid.New()
	require.NoError(t, store.Schedule(ctx, due, time.Now().Add(-time.Second)))
	require.NoError(t, store.Schedule(ctx, future, time.Now().Add(time.Hour)))

	popped, err := store.PopDue(ctx, 10)
	require.NoError(t, err)
	require.ElementsMatch(t, []uuid.UUID{due}, popped)
}

// TestTripOfferTimeout_PopDue_RemovesClaimed: un trip_id reclamado no puede
// volver a salir en un barrido posterior, ni quedar en la cola.
func TestTripOfferTimeout_PopDue_RemovesClaimed(t *testing.T) {
	store := repo.NewTripOfferTimeoutRepo(newOfferTimeoutStoreForTest(t))
	ctx := context.Background()

	tripID := uuid.New()
	require.NoError(t, store.Schedule(ctx, tripID, time.Now().Add(-time.Second)))

	first, err := store.PopDue(ctx, 10)
	require.NoError(t, err)
	require.Equal(t, []uuid.UUID{tripID}, first)

	second, err := store.PopDue(ctx, 10)
	require.NoError(t, err)
	require.Empty(t, second)
}

// TestTripOfferTimeout_PopDue_ExactlyOnceAcrossReplicas simula varias
// réplicas reclamando el mismo vencimiento al mismo tiempo (ver "Hecho
// cuando" de #359: con dos réplicas, un timeout se procesa una sola vez).
func TestTripOfferTimeout_PopDue_ExactlyOnceAcrossReplicas(t *testing.T) {
	client := newOfferTimeoutStoreForTest(t)
	store := repo.NewTripOfferTimeoutRepo(client)
	ctx := context.Background()

	const tripCount = 50
	tripIDs := make([]uuid.UUID, tripCount)
	for i := range tripIDs {
		tripIDs[i] = uuid.New()
		require.NoError(t, store.Schedule(ctx, tripIDs[i], time.Now().Add(-time.Second)))
	}

	const replicas = 8
	var mu sync.Mutex
	claimed := make(map[uuid.UUID]int)
	var wg sync.WaitGroup
	for range replicas {
		wg.Go(func() {
			// Cada "réplica" comparte el mismo cliente Redis, como pasaría en
			// producción: lo que las aísla es el script atómico, no la conexión.
			replicaStore := repo.NewTripOfferTimeoutRepo(client)
			for range tripCount {
				popped, err := replicaStore.PopDue(ctx, 1)
				if err != nil {
					continue
				}
				mu.Lock()
				for _, id := range popped {
					claimed[id]++
				}
				mu.Unlock()
			}
		})
	}
	wg.Wait()

	require.Len(t, claimed, tripCount, "todos los vencimientos debieron reclamarse")
	for id, count := range claimed {
		require.Equal(t, 1, count, "trip %s se reclamó más de una vez", id)
	}
}

// TestTripOfferTimeout_Cancel_RemovesFromQueue cubre que un viaje aceptado o
// cancelado no dispare su ronda: Cancel saca el vencimiento aunque todavía no
// esté vencido.
func TestTripOfferTimeout_Cancel_RemovesFromQueue(t *testing.T) {
	store := repo.NewTripOfferTimeoutRepo(newOfferTimeoutStoreForTest(t))
	ctx := context.Background()

	tripID := uuid.New()
	require.NoError(t, store.Schedule(ctx, tripID, time.Now().Add(-time.Second)))
	require.NoError(t, store.Cancel(ctx, tripID))

	popped, err := store.PopDue(ctx, 10)
	require.NoError(t, err)
	require.Empty(t, popped)
}

// TestTripOfferTimeout_ScheduleIfAbsent_DoesNotOverwritePending es la base
// del barrido de rescate: no debe adelantar el vencimiento de un viaje que ya
// está esperando su ronda real más adelante en el tiempo.
func TestTripOfferTimeout_ScheduleIfAbsent_DoesNotOverwritePending(t *testing.T) {
	store := repo.NewTripOfferTimeoutRepo(newOfferTimeoutStoreForTest(t))
	ctx := context.Background()

	tripID := uuid.New()
	future := time.Now().Add(time.Hour)
	require.NoError(t, store.Schedule(ctx, tripID, future))

	// El rescate intenta programarlo "ahora": no debe pisar el score futuro.
	require.NoError(t, store.ScheduleIfAbsent(ctx, tripID, time.Now()))

	popped, err := store.PopDue(ctx, 10)
	require.NoError(t, err)
	require.Empty(t, popped, "ScheduleIfAbsent no debió adelantar un vencimiento ya programado")
}

// TestTripOfferTimeout_ScheduleIfAbsent_RescuesOrphan es el caso que #359
// pide explícitamente: un viaje sin vencimiento en cola (huérfano de un
// reinicio) sí debe quedar listo para que el próximo barrido lo procese.
func TestTripOfferTimeout_ScheduleIfAbsent_RescuesOrphan(t *testing.T) {
	store := repo.NewTripOfferTimeoutRepo(newOfferTimeoutStoreForTest(t))
	ctx := context.Background()

	tripID := uuid.New()
	require.NoError(t, store.ScheduleIfAbsent(ctx, tripID, time.Now()))

	popped, err := store.PopDue(ctx, 10)
	require.NoError(t, err)
	require.Equal(t, []uuid.UUID{tripID}, popped)
}
