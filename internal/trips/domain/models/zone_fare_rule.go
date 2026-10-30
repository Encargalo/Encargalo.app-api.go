package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// ZoneFareRule es la mínima y la tarifa extra/km por banda horaria de una
// zona geográfica (ej. Charallave), que remplaza la fórmula global de
// tripfare.go para los viajes cuyo pickup cae en una de sus celdas.
// A diferencia del multiplicador global de banda, estos montos ya son
// finales por banda: no se multiplican por nada más.
// IncludedKm es un solo valor por zona: las tres bandas lo comparten.
type ZoneFareRule struct {
	bun.BaseModel `bun:"table:trips.zone_fare_rules"`

	ZoneID     uuid.UUID `bun:"zone_id,pk,type:uuid"`
	Name       string    `bun:"name,notnull"`
	IncludedKm float64   `bun:"included_km,notnull"`

	DayMinFareUSD                 float64 `bun:"day_min_fare_usd,notnull"`
	DayExtraRatePerKmUSD          float64 `bun:"day_extra_rate_per_km_usd,notnull"`
	NightMinFareUSD               float64 `bun:"night_min_fare_usd,notnull"`
	NightExtraRatePerKmUSD        float64 `bun:"night_extra_rate_per_km_usd,notnull"`
	EarlyMorningMinFareUSD        float64 `bun:"early_morning_min_fare_usd,notnull"`
	EarlyMorningExtraRatePerKmUSD float64 `bun:"early_morning_extra_rate_per_km_usd,notnull"`

	// Active en false es un soft delete: FareQuote ya pudo congelar este
	// ZoneID, así que borrarlo de verdad rompería su trazabilidad histórica.
	Active    bool      `bun:"active,notnull"`
	CreatedAt time.Time `bun:"created_at,notnull"`
	UpdatedAt time.Time `bun:"updated_at,notnull"`
}

// ZoneFareCell mapea una celda H3 (resolución geo.ZoneAnalyticsResolution) a
// la zona cuya regla aplica cuando el pickup de un viaje cae ahí. Una celda
// pertenece a una sola zona.
type ZoneFareCell struct {
	bun.BaseModel `bun:"table:trips.zone_fare_cells"`

	H3Cell string    `bun:"h3_cell,pk"`
	ZoneID uuid.UUID `bun:"zone_id,type:uuid,notnull"`
}
