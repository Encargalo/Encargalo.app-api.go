package app

import (
	"context"
	"log/slog"
	"time"

	paymentsPorts "Encargalo.app-api.go/internal/payments/domain/ports"
	"Encargalo.app-api.go/internal/trips/domain/models"
	usersModels "Encargalo.app-api.go/internal/users/domain/models"
	"github.com/google/uuid"
)

// CreditUnservedPagoMovil acredita fare × bcv_rate a la cartera cuando el
// Pago Móvil ya estaba aprobado y el viaje no se prestó. Un segundo llamado
// no vuelve a acreditar: PagoMovilChargeBs falla si ya hay transaction_id.
func CreditUnservedPagoMovil(
	ctx context.Context,
	balance paymentsPorts.CustomerBalanceCreditor,
	trip *models.Trip,
	save func(context.Context, uuid.UUID, *models.TripPayment) error,
) {
	if balance == nil || trip == nil {
		return
	}
	amount, rate, ok := trip.PagoMovilChargeBs()
	if !ok {
		return
	}
	txID, createdAt, err := balance.CreditOrderRejected(ctx, trip.CustomerID, trip.ID, models.ReferenceTypeTrip, amount, amount, rate)
	if err != nil {
		slog.Error("credit pago movil on unserved trip", "tripID", trip.ID, "error", err)
		return
	}
	var until *time.Time
	if usersModels.MeetsRefundMin(amount, rate) {
		u := createdAt.Add(usersModels.RefundRequestWindow)
		until = &u
	}
	trip.RecordPagoMovilRefund(txID, amount, until)
	if save == nil || trip.Payment == nil {
		return
	}
	if err := save(ctx, trip.ID, trip.Payment); err != nil {
		slog.Error("save trip payment refund", "tripID", trip.ID, "error", err)
	}
}
