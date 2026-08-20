package app

import (
	"time"

	"Encargalo.app-api.go/internal/orders/domain/models"
	usersModels "Encargalo.app-api.go/internal/users/domain/models"
	"github.com/google/uuid"
)

// applyBalanceCreditToPaymentDetails rellena en details.payment los campos para
// que el cliente pueda solicitar devolución desde el order_update WS (misma
// info que GET /users/balance sobre el movimiento cash-refundable). Recibe
// los montos ya en Bs (moneda nativa del balance) y bcvRate congelada para
// el umbral RefundMinAmountUSD, igual que usersModels.MeetsRefundMin.
func applyBalanceCreditToPaymentDetails(
	pd *models.PaymentDetails,
	txID uuid.UUID,
	creditedBs, refundablePMBs, bcvRate float64,
	createdAt time.Time,
) {
	if pd == nil || txID == uuid.Nil {
		return
	}
	id := txID
	pd.TransactionID = &id
	if creditedBs > 0 {
		c := creditedBs
		pd.CreditedBs = &c
	}
	if !usersModels.MeetsRefundMin(refundablePMBs, bcvRate) {
		return
	}
	r := refundablePMBs
	pd.RefundablePMBs = &r
	until := createdAt.Add(usersModels.RefundRequestWindow)
	untilStr := until.Format(time.RFC3339)
	pd.RefundEligibleUntil = &untilStr
	if !time.Now().After(until) {
		pd.CanRequestRefund = true
	}
}
