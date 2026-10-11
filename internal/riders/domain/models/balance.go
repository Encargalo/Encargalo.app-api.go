package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type MovementType string
type Zone string

const (
	MovementTypeRideBank MovementType = "ride_bank"
	MovementTypeRideCash MovementType = "ride_cash"
	MovementTypeTripBank MovementType = "trip_bank"
	MovementTypeTripCash MovementType = "trip_cash"
	MovementTypeWithdraw MovementType = "withdrawal"

	// PaymentMethodEfectivo es el method_payment que cobra el rider en la entrega.
	PaymentMethodEfectivo = "Efectivo"

	MovementTypeStaffPayout      MovementType = "staff_payout"
	MovementTypeAdjustmentCredit MovementType = "adjustment_credit"
	MovementTypeAdjustmentDebit  MovementType = "adjustment_debit"

	ZoneNormal              Zone = "normal"
	ZoneWithdrawalAvailable Zone = "withdrawal_available"

	// WithdrawalMinUSD es el piso de negocio, expresado en USD porque es la
	// unidad estable de referencia. El mínimo real en Bs se recalcula en cada
	// consulta con la tasa BCV vigente (ver riders/app/finance.go), para que
	// el poder adquisitivo del piso no se erosione con la tasa.
	WithdrawalMinUSD   float64 = 15.0
	WithdrawalCooldown         = 24 * time.Hour
	RiderShare         float64 = 0.90

	// CurrencyVES es la moneda nativa del balance de riders: cada movimiento
	// se registra ya convertido a bolívares con la tasa BCV vigente al
	// momento del movimiento (ver Transaction.BcvRate).
	CurrencyVES = "VES"

	// ServiceTypeDelivery y ServiceTypeTrip distinguen, en movimientos
	// ride_bank/trip_bank, si la ganancia del rider vino de un domicilio o de
	// un viaje (ambos comparten el mismo ledger de balance/transactions).
	ServiceTypeDelivery = "delivery"
	ServiceTypeTrip     = "trip"
)

type Balance struct {
	bun.BaseModel `bun:"table:finance.balances"`

	EntityID   uuid.UUID `bun:"entity_id,pk,type:uuid"`
	EntityType string    `bun:"entity_type,pk"`
	Amount     float64   `bun:"amount"`
	Currency   string    `bun:"currency"`
	UpdatedAt  time.Time `bun:"updated_at"`
}

type Transaction struct {
	bun.BaseModel `bun:"table:finance.transactions"`

	ID         uuid.UUID  `bun:"id,pk,type:uuid"`
	EntityID   uuid.UUID  `bun:"entity_id"`
	EntityType string     `bun:"entity_type"`
	OrderID    *uuid.UUID `bun:"order_id,nullzero"`
	TripID     *uuid.UUID `bun:"trip_id,nullzero"`
	Amount     float64    `bun:"amount"`
	Currency   string     `bun:"currency"`
	// BcvRate es la tasa usada para convertir el monto original en USD
	// (delivery fee) al Bs acreditado en Amount. Nil en movimientos que ya
	// nacen en Bs y no requieren conversión (ej. withdrawal).
	BcvRate       *float64   `bun:"bcv_rate,nullzero"`
	MovementType  string     `bun:"movement_type"`
	ServiceType   *string    `bun:"service_type,nullzero"`
	PaymentMethod *string    `bun:"payment_method"`
	DistanceKm    *float64   `bun:"distance_km"`
	StaffID       *uuid.UUID `bun:"staff_id,nullzero"`
	Notes         *string    `bun:"notes,nullzero"`
	ProofURL      *string    `bun:"proof_url,nullzero"`
	CreatedAt     time.Time  `bun:"created_at,nullzero"`
}

// RiderPayUSD es la parte del rider (90%) de una tarifa de envío o de viaje.
func RiderPayUSD(fee float64) float64 {
	return fee * RiderShare
}

// CashOwedUSD es lo que el rider nos debe después de quedarse con su pago.
// Ese pago ya viene dentro del efectivo que cobró: un pedido de $10 cuyo
// pago es $2 deja una deuda de $8, no de $10 ni de $2.
func CashOwedUSD(collected, riderPay float64) float64 {
	owed := collected - riderPay
	if owed < 0 {
		return 0
	}
	return owed
}
