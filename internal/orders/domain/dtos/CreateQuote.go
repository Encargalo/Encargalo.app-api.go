package dtos

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type CreateQuote struct {
	ShopID     uuid.UUID          `json:"shop_id" validate:"required,uuid"`
	CustomerID uuid.UUID          `json:"customer_id" swaggerignore:"true"`
	Address    Address            `json:"address" validate:"required"`
	Items      []CreateItemsOrder `json:"items" validate:"required,dive,required"`
}

func (q *CreateQuote) Validate() error {
	_ = conform.Struct(context.Background(), q)
	return validate.Struct(q)
}

type QuoteResponse struct {
	QuoteID        uuid.UUID `json:"quote_id"`
	TotalUSD       float64   `json:"total_usd"`
	TotalBs        float64   `json:"total_bs"`
	DeliveryFeeUSD float64   `json:"delivery_fee_usd"`
	DeliveryFeeBs  float64   `json:"delivery_fee_bs"`
	// DeliveryFeeOriginalUSD y DeliverySubsidyUSD solo vienen con el envío gratis de primer pedido; delivery_fee_usd ya lo trae restado.
	DeliveryFeeOriginalUSD float64         `json:"delivery_fee_original_usd,omitempty"`
	DeliverySubsidyUSD     float64         `json:"delivery_subsidy_usd,omitempty"`
	BcvRate                float64         `json:"bcv_rate"`
	IsLaunchPromo          bool            `json:"is_launch_promo"`
	PromoSubsidyAmount     float64         `json:"promo_subsidy_amount,omitempty"`
	ExpiresAt              time.Time       `json:"expires_at"`
	ProofGraceUntil        time.Time       `json:"proof_grace_until"`
	Items                  []QuoteItemView `json:"items"`
}

type QuoteItemView struct {
	ItemID     uuid.UUID `json:"item_id"`
	Name       string    `json:"name"`
	Amount     int       `json:"amount"`
	UnitPrice  float64   `json:"unit_price"`
	TotalPrice float64   `json:"total_price"`
}
