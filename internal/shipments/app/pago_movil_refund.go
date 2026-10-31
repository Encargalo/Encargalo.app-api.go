package app

import (
	"context"
	"log/slog"
	"time"

	paymentsPorts "Encargalo.app-api.go/internal/payments/domain/ports"
	"Encargalo.app-api.go/internal/shipments/domain/models"
	usersModels "Encargalo.app-api.go/internal/users/domain/models"
	"github.com/google/uuid"
)

// CreditUnservedPagoMovil acredita fare × bcv_rate a la cartera cuando el
// Pago Móvil ya estaba aprobado y el envío no se prestó. Un segundo llamado
// no vuelve a acreditar: PagoMovilChargeBs falla si ya hay transaction_id.
func CreditUnservedPagoMovil(
	ctx context.Context,
	balance paymentsPorts.CustomerBalanceCreditor,
	shipment *models.Shipment,
	save func(context.Context, uuid.UUID, *models.ShipmentPayment) error,
) {
	if balance == nil || shipment == nil {
		return
	}
	amount, rate, ok := shipment.PagoMovilChargeBs()
	if !ok {
		return
	}
	txID, createdAt, err := balance.CreditOrderRejected(ctx, shipment.CustomerID, shipment.ID, models.ReferenceTypeShipment, amount, amount, rate)
	if err != nil {
		slog.Error("credit pago movil on unserved shipment", "shipmentID", shipment.ID, "error", err)
		return
	}
	var until *time.Time
	if usersModels.MeetsRefundMin(amount, rate) {
		u := createdAt.Add(usersModels.RefundRequestWindow)
		until = &u
	}
	shipment.RecordPagoMovilRefund(txID, amount, until)
	if save == nil || shipment.Payment == nil {
		return
	}
	if err := save(ctx, shipment.ID, shipment.Payment); err != nil {
		slog.Error("save shipment payment refund", "shipmentID", shipment.ID, "error", err)
	}
}
