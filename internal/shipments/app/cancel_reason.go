package app

import (
	"context"
	"log/slog"

	"Encargalo.app-api.go/internal/shipments/domain/models"
	"github.com/google/uuid"
)

// saveCancelReason deja el motivo en el envío y en la base. Un fallo al
// persistir no debe tumbar la cancelación: el estado ya cambió.
func saveCancelReason(
	ctx context.Context,
	shipment *models.Shipment,
	reason models.CancelReason,
	save func(context.Context, uuid.UUID, models.CancelReason) error,
) {
	if shipment == nil || shipment.CancelReason != nil {
		return
	}
	shipment.SetCancelReason(reason)
	if save == nil {
		return
	}
	if err := save(ctx, shipment.ID, reason); err != nil {
		slog.Error("save shipment cancel reason", "shipmentID", shipment.ID, "reason", reason, "error", err)
	}
}
