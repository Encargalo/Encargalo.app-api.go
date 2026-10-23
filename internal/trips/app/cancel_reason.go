package app

import (
	"context"
	"log/slog"

	"Encargalo.app-api.go/internal/trips/domain/models"
	"github.com/google/uuid"
)

// saveCancelReason deja el motivo en el viaje y en la base. Un fallo al
// persistir no debe tumbar la cancelación: el estado ya cambió.
func saveCancelReason(
	ctx context.Context,
	trip *models.Trip,
	reason models.CancelReason,
	save func(context.Context, uuid.UUID, models.CancelReason) error,
) {
	if trip == nil || trip.CancelReason != nil {
		return
	}
	trip.SetCancelReason(reason)
	if save == nil {
		return
	}
	if err := save(ctx, trip.ID, reason); err != nil {
		slog.Error("save trip cancel reason", "tripID", trip.ID, "reason", reason, "error", err)
	}
}
