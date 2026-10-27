package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// OrderFareQuote es el registro durable de una cotización de delivery, para
// medir conversión. La OrderQuote de Redis se borra al consumirse y expira,
// así que sin esta fila una cotización que no convirtió no deja rastro.
type OrderFareQuote struct {
	bun.BaseModel `bun:"table:orders.fare_quotes"`

	QuoteID            uuid.UUID `bun:"quote_id,pk,type:uuid"`
	CustomerID         uuid.UUID `bun:"customer_id,type:uuid,notnull"`
	ShopID             uuid.UUID `bun:"shop_id,type:uuid,notnull"`
	ShopLatitude       float64   `bun:"shop_latitude,notnull"`
	ShopLongitude      float64   `bun:"shop_longitude,notnull"`
	DeliveryLatitude   float64   `bun:"delivery_latitude,notnull"`
	DeliveryLongitude  float64   `bun:"delivery_longitude,notnull"`
	DistanceMeters     float64   `bun:"distance_meters,notnull"`
	FeeUSD             float64   `bun:"fee_usd,notnull"`
	FeeBs              float64   `bun:"fee_bs,notnull"`
	DeliverySubsidyUSD float64   `bun:"delivery_subsidy_usd,notnull,default:0"`
	BcvRate            float64   `bun:"bcv_rate,notnull"`
	FareBand           string    `bun:"fare_band,notnull"`
	CreatedAt          time.Time `bun:"created_at,notnull"`
}
