package app

import (
	"context"
	"log/slog"
	"time"

	"Encargalo.app-api.go/internal/riders/domain/ports"
)

const (
	// tripOfferTimeoutSweepInterval es cada cuánto se reclaman vencimientos
	// ya cumplidos. No necesita ser más fino que esto: la ronda más corta
	// (tripEmptyRoundTimeout) dura minutos.
	tripOfferTimeoutSweepInterval = 5 * time.Second
	tripOfferTimeoutBatchSize     = 50
	// tripOfferRescueInterval es cada cuánto se buscan viajes en "searching"
	// sin vencimiento en cola (ver #359): un reinicio pasado, o una carrera
	// entre guardar el viaje y el ZADD, los puede dejar así.
	tripOfferRescueInterval = 30 * time.Second
)

// TripOfferTimeoutWorker reemplaza los time.AfterFunc en memoria de las
// rondas de oferta (ver #359): el vencimiento vive en Redis via
// ports.TripOfferTimeoutStore, así que un reinicio del proceso no deja
// viajes colgados en "searching".
type TripOfferTimeoutWorker struct {
	mgr   *appRidersManager
	queue ports.TripOfferTimeoutStore
	trips ports.TripsStore
}

func NewTripOfferTimeoutWorker(svc ports.RidersApp, queue ports.TripOfferTimeoutStore, trips ports.TripsStore) *TripOfferTimeoutWorker {
	return &TripOfferTimeoutWorker{mgr: svc.(*appRidersManager), queue: queue, trips: trips}
}

func (w *TripOfferTimeoutWorker) Start(ctx context.Context) {
	go w.loop(ctx)
}

func (w *TripOfferTimeoutWorker) loop(ctx context.Context) {
	dueTicker := time.NewTicker(tripOfferTimeoutSweepInterval)
	defer dueTicker.Stop()
	rescueTicker := time.NewTicker(tripOfferRescueInterval)
	defer rescueTicker.Stop()

	// Primeras pasadas al arrancar: recuperan lo que quedó colgado en un reinicio.
	w.processDue(ctx)
	w.rescueOrphans(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-dueTicker.C:
			w.processDue(ctx)
		case <-rescueTicker.C:
			w.rescueOrphans(ctx)
		}
	}
}

func (w *TripOfferTimeoutWorker) processDue(ctx context.Context) {
	due, err := w.queue.PopDue(ctx, tripOfferTimeoutBatchSize)
	if err != nil {
		slog.Error("TripOfferTimeoutWorker PopDue failed", "error", err)
		return
	}
	for _, tripID := range due {
		w.mgr.onTripOfferTimeout(ctx, tripID)
	}
}

// rescueOrphans programa un vencimiento inmediato para cualquier viaje en
// "searching" que no tenga uno en cola. ScheduleIfAbsent es un ZADD NX: no
// toca a los que sí están esperando su ronda real.
func (w *TripOfferTimeoutWorker) rescueOrphans(ctx context.Context) {
	searching, err := w.trips.ListSearchingTripIDs(ctx)
	if err != nil {
		slog.Error("TripOfferTimeoutWorker rescue list failed", "error", err)
		return
	}
	now := time.Now()
	for _, tripID := range searching {
		if err := w.queue.ScheduleIfAbsent(ctx, tripID, now); err != nil {
			slog.Error("TripOfferTimeoutWorker rescue schedule failed", "tripID", tripID, "error", err)
		}
	}
}
