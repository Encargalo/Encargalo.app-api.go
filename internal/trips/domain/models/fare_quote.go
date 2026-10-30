package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// FareQuote es el registro durable de una cotización, para medir conversión.
// La TripQuote de Redis se borra al consumirse y expira a los 10 minutos, así
// que sin esta fila una cotización que no convirtió no deja rastro.
type FareQuote struct {
	bun.BaseModel `bun:"table:trips.fare_quotes"`

	QuoteID           uuid.UUID `bun:"quote_id,pk,type:uuid"`
	CustomerID        uuid.UUID `bun:"customer_id,type:uuid,notnull"`
	PickupLatitude    float64   `bun:"pickup_latitude,notnull"`
	PickupLongitude   float64   `bun:"pickup_longitude,notnull"`
	DropoffLatitude   float64   `bun:"dropoff_latitude,notnull"`
	DropoffLongitude  float64   `bun:"dropoff_longitude,notnull"`
	PickupH3CellZone  *string   `bun:"pickup_h3_cell_zone"`
	DropoffH3CellZone *string   `bun:"dropoff_h3_cell_zone"`
	DistanceMeters    float64   `bun:"distance_meters,notnull"`
	DurationMin       int64     `bun:"duration_min,notnull"`
	FareUSD           float64   `bun:"fare_usd,notnull"`
	FareBs            float64   `bun:"fare_bs,notnull"`
	DiscountUSD       float64   `bun:"discount_usd,notnull,default:0"`
	BcvRate           float64   `bun:"bcv_rate,notnull"`
	FareBand          string    `bun:"fare_band,notnull"`
	// ZoneFareRuleID es la zona cuya regla produjo FareUSD, congelada igual
	// que FareBand/BcvRate. nil = se usó el fallback global de tripfare.go.
	ZoneFareRuleID *uuid.UUID `bun:"zone_fare_rule_id"`
	VehicleType    string     `bun:"vehicle_type,notnull"`
	CreatedAt      time.Time  `bun:"created_at,notnull"`
}
