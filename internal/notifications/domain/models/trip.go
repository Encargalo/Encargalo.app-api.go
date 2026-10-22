package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Trip struct {
	bun.BaseModel  `bun:"table:trips.trips"`
	ID             uuid.UUID `bun:"id,pk"`
	CustomerID     uuid.UUID `bun:"customer_id"`
	Status         string    `bun:"status"`
	PickupAddress  string    `bun:"pickup_address"`
	DropoffAddress string    `bun:"dropoff_address"`
	Fare           *float64  `bun:"fare"`
	BcvRate        *float64  `bun:"bcv_rate"`
	MethodPayment  string    `bun:"method_payment"`
	BalancePaidBs  *float64  `bun:"balance_paid_bs"`
	CashAmountBs   *float64  `bun:"cash_amount_bs"`
	RequestedAt    time.Time `bun:"requested_at"`

	Customer Customer `bun:"rel:belongs-to,join:customer_id=id"`
}

// ShortID identifica el viaje en Slack; los viajes no tienen número correlativo.
func (t Trip) ShortID() string {
	return "#" + t.ID.String()[:8]
}

func (t Trip) FareUSD() float64 {
	if t.Fare == nil {
		return 0
	}
	return *t.Fare
}

// Rate es la tasa BCV congelada en la cotización del viaje.
func (t Trip) Rate() float64 {
	if t.BcvRate == nil {
		return 0
	}
	return *t.BcvRate
}

func (t Trip) PaymentLabel() string {
	label := paymentMethodLabel(t.MethodPayment)
	if t.MethodPayment != "Balance" && t.BalancePaidBs != nil && *t.BalancePaidBs > 0 {
		label += " + Saldo"
	}
	if t.MethodPayment != "Efectivo" && t.CashAmountBs != nil && *t.CashAmountBs > 0 {
		label += " + Efectivo"
	}
	return label
}

func paymentMethodLabel(method string) string {
	switch method {
	case "PagoMovil":
		return "Pago Móvil"
	case "Balance":
		return "Saldo"
	case "Efectivo":
		return "Efectivo"
	case "":
		return "—"
	default:
		return method
	}
}
