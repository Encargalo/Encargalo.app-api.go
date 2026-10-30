package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

const (
	EntityTypeShop            = "shop"
	EntityTypePlatformSubsidy = "platform_subsidy"
	CurrencyUSD               = "USD"
	ShopShare                 = 0.96
	// ZeroCommissionShare es el share que recibe el negocio durante sus
	// primeros 8 días de vida (comisión 0% en vez del 4% habitual).
	ZeroCommissionShare = 1.0
	MethodEfectivo      = "Efectivo"
)

// ShopSaleShare devuelve el porcentaje del subtotal que recibe el negocio.
func ShopSaleShare(isZeroCommission bool) float64 {
	if isZeroCommission {
		return ZeroCommissionShare
	}
	return ShopShare
}

// SaleSubtotal reconstruye el subtotal sobre el que se liquida al negocio.
// Con promo de lanzamiento, promoSubsidyAmount suma de vuelta lo que el
// cliente no pagó para que el crédito sea sobre el precio de catálogo.
func SaleSubtotal(totalPrice, deliveryFee, promoSubsidyAmount float64, isLaunchPromo bool) float64 {
	base := totalPrice - deliveryFee
	if isLaunchPromo {
		base += promoSubsidyAmount
	}
	return base
}

// ShopSaleBreakdown desglosa base, comisión y neto de una venta al negocio.
type ShopSaleBreakdown struct {
	Base       float64
	Share      float64
	Commission float64
	Net        float64
}

// ComputeShopSaleBreakdown es la única fuente de verdad para liquidación WS
// (orders/managerOrder) y RecordSaleCommission (shops/finance).
func ComputeShopSaleBreakdown(totalPrice, deliveryFee, promoSubsidyAmount float64, isLaunchPromo, isZeroCommission bool) ShopSaleBreakdown {
	base := SaleSubtotal(totalPrice, deliveryFee, promoSubsidyAmount, isLaunchPromo)
	share := ShopSaleShare(isZeroCommission)
	net := base * share
	return ShopSaleBreakdown{
		Base:       base,
		Share:      share,
		Commission: base - net,
		Net:        net,
	}
}

type ShopMovementType string

const (
	MovementTypeSaleDeliveryBank   ShopMovementType = "sale_delivery_bank"
	MovementTypeSaleDeliveryCash   ShopMovementType = "sale_delivery_cash"
	MovementTypeLaunchPromoSubsidy ShopMovementType = "launch_promo_subsidy"
	MovementTypeStaffPayout        ShopMovementType = "staff_payout"
	MovementTypeAdjustmentCredit   ShopMovementType = "adjustment_credit"
	MovementTypeAdjustmentDebit    ShopMovementType = "adjustment_debit"
)

// SaleMovementTypes son los movimientos que cuentan como ingreso por ventas en estadísticas.
var SaleMovementTypes = []string{
	string(MovementTypeSaleDeliveryBank),
	string(MovementTypeSaleDeliveryCash),
	string(MovementTypeLaunchPromoSubsidy),
}

type ShopBalance struct {
	bun.BaseModel `bun:"table:finance.balances"`

	EntityID   uuid.UUID `bun:"entity_id,pk,type:uuid"`
	EntityType string    `bun:"entity_type,pk"`
	Amount     float64   `bun:"amount"`
	Currency   string    `bun:"currency"`
	UpdatedAt  time.Time `bun:"updated_at"`
}

type ShopTransaction struct {
	bun.BaseModel `bun:"table:finance.transactions"`

	ID            uuid.UUID  `bun:"id,pk,type:uuid"`
	EntityID      uuid.UUID  `bun:"entity_id"`
	EntityType    string     `bun:"entity_type"`
	OrderID       *uuid.UUID `bun:"order_id,nullzero"`
	Amount        float64    `bun:"amount"`
	Currency      string     `bun:"currency"`
	BcvRate       *float64   `bun:"bcv_rate,nullzero"`
	MovementType  string     `bun:"movement_type"`
	PaymentMethod *string    `bun:"payment_method"`
	StaffID       *uuid.UUID `bun:"staff_id,nullzero"`
	Notes         *string    `bun:"notes,nullzero"`
	ProofURL      *string    `bun:"proof_url,nullzero"`
	CreatedAt     time.Time  `bun:"created_at,nullzero"`
}

type CustomerMetricsRow struct {
	NewCustomers       int `bun:"new_customers"`
	ReturningCustomers int `bun:"returning_customers"`
}
