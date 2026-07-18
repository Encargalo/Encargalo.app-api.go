package dtos

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type CreateQuote struct {
	ShopID     uuid.UUID          `json:"shop_id" validate:"required,uuid4"`
	CustomerID uuid.UUID          `json:"customer_id" swaggerignore:"true"`
	Address    Address            `json:"address" validate:"required"`
	Items      []CreateItemsOrder `json:"items" validate:"required,dive,required"`
}

func (q *CreateQuote) Validate() error {
	_ = conform.Struct(context.Background(), q)
	return validate.Struct(q)
}

type QuoteResponse struct {
	QuoteID            uuid.UUID       `json:"quote_id"`
	TotalUSD           float64         `json:"total_usd"`
	TotalBs            float64         `json:"total_bs"`
	DeliveryFeeUSD     float64         `json:"delivery_fee_usd"`
	DeliveryFeeBs      float64         `json:"delivery_fee_bs"`
	BcvRate            float64         `json:"bcv_rate"`
	IsLaunchPromo      bool            `json:"is_launch_promo"`
	PromoSubsidyAmount float64         `json:"promo_subsidy_amount,omitempty"`
	ExpiresAt          time.Time       `json:"expires_at"`
	ProofGraceUntil    time.Time       `json:"proof_grace_until"`
	Items              []QuoteItemView `json:"items"`
}

type QuoteItemView struct {
	ItemID     uuid.UUID `json:"item_id"`
	Name       string    `json:"name"`
	Amount     int       `json:"amount"`
	UnitPrice  int       `json:"unit_price"`
	TotalPrice int       `json:"total_price"`
}
