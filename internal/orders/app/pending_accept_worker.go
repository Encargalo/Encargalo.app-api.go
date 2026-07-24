package app

import (
	"context"
	"log/slog"
	"time"

	"Encargalo.app-api.go/internal/orders/domain/ports"
)

const pendingAcceptWorkerInterval = 15 * time.Second

// PendingAcceptWorker rechaza órdenes Pending sin aceptación del local
// tras models.PendingAcceptTimeout.
type PendingAcceptWorker struct {
	orders ports.OrdersApp
	mgr    ports.AppManagerOrder
}

func NewPendingAcceptWorker(orders ports.OrdersApp, mgr ports.AppManagerOrder) *PendingAcceptWorker {
	return &PendingAcceptWorker{orders: orders, mgr: mgr}
}

func (w *PendingAcceptWorker) Start(ctx context.Context) {
	go w.loop(ctx)
}

func (w *PendingAcceptWorker) loop(ctx context.Context) {
	ticker := time.NewTicker(pendingAcceptWorkerInterval)
	defer ticker.Stop()

	// Primera pasada al arrancar (por si el server estuvo caído).
	w.tick(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.tick(ctx)
		}
	}
}

func (w *PendingAcceptWorker) tick(ctx context.Context) {
	if err := w.orders.NotifyPendingUnassigned(ctx); err != nil {
		slog.Error("PendingAcceptWorker notify unassigned failed", "error", err)
	}

	rejected, err := w.orders.AutoRejectStalePending(ctx)
	if err != nil {
		slog.Error("PendingAcceptWorker tick failed", "error", err)
		return
	}
	if w.mgr == nil {
		return
	}
	for i := range rejected {
		if err := w.mgr.BroadcastOrder(rejected[i]); err != nil {
			slog.Error("PendingAcceptWorker broadcast failed",
				"orderID", rejected[i].ID, "error", err)
		}
	}
}
