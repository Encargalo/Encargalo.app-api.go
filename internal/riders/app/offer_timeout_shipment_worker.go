package app

import (
	"context"
	"log/slog"
	"time"

	"Encargalo.app-api.go/internal/riders/domain/ports"
)

const (
	// shipmentOfferTimeoutSweepInterval: mismo intervalo que trips, no necesita
	// ser más fino — el único vencimiento (shipmentOfferTimeout) dura minutos.
	shipmentOfferTimeoutSweepInterval = 5 * time.Second
	shipmentOfferTimeoutBatchSize     = 50
	// shipmentOfferRescueInterval reprograma envíos en "searching" sin
	// vencimiento en cola, análogo a tripOfferRescueInterval.
	shipmentOfferRescueInterval = 30 * time.Second
)

// ShipmentOfferTimeoutWorker es el análogo simplificado (sin rondas) de
// TripOfferTimeoutWorker: el vencimiento del broadcast de un envío vive en
// Redis via ports.ShipmentOfferTimeoutStore, así que un reinicio del proceso
// no deja envíos colgados en "searching".
type ShipmentOfferTimeoutWorker struct {
	mgr       *appRidersManager
	queue     ports.ShipmentOfferTimeoutStore
	shipments ports.ShipmentsStore
}

func NewShipmentOfferTimeoutWorker(svc ports.RidersApp, queue ports.ShipmentOfferTimeoutStore, shipments ports.ShipmentsStore) *ShipmentOfferTimeoutWorker {
	return &ShipmentOfferTimeoutWorker{mgr: svc.(*appRidersManager), queue: queue, shipments: shipments}
}

func (w *ShipmentOfferTimeoutWorker) Start(ctx context.Context) {
	go w.loop(ctx)
}

func (w *ShipmentOfferTimeoutWorker) loop(ctx context.Context) {
	dueTicker := time.NewTicker(shipmentOfferTimeoutSweepInterval)
	defer dueTicker.Stop()
	rescueTicker := time.NewTicker(shipmentOfferRescueInterval)
	defer rescueTicker.Stop()

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

func (w *ShipmentOfferTimeoutWorker) processDue(ctx context.Context) {
	due, err := w.queue.PopDue(ctx, shipmentOfferTimeoutBatchSize)
	if err != nil {
		slog.Error("ShipmentOfferTimeoutWorker PopDue failed", "error", err)
		return
	}
	for _, shipmentID := range due {
		w.mgr.onShipmentOfferTimeout(ctx, shipmentID)
	}
}

// rescueOrphans programa un vencimiento inmediato para cualquier envío en
// "searching" sin uno en cola — ScheduleIfAbsent es un ZADD NX, análogo a
// TripOfferTimeoutWorker.rescueOrphans.
func (w *ShipmentOfferTimeoutWorker) rescueOrphans(ctx context.Context) {
	searching, err := w.shipments.ListSearchingShipmentIDs(ctx)
	if err != nil {
		slog.Error("ShipmentOfferTimeoutWorker rescue list failed", "error", err)
		return
	}
	now := time.Now()
	for _, shipmentID := range searching {
		if err := w.queue.ScheduleIfAbsent(ctx, shipmentID, now); err != nil {
			slog.Error("ShipmentOfferTimeoutWorker rescue schedule failed", "shipmentID", shipmentID, "error", err)
		}
	}
}
