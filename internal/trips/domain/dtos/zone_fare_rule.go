package dtos

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// LatLngPoint es un punto del polígono que el CRM dibuja o pega (ej. desde
// geojson.io) para delimitar una zona de tarifa.
type LatLngPoint struct {
	Lat float64 `json:"lat" validate:"required,latitude" example:"10.1167"`
	Lng float64 `json:"lng" validate:"required,longitude" example:"-66.8500"`
}

// ZoneFareRuleRequest crea o reemplaza por completo una zona de tarifa: el
// polígono se vuelve a convertir a celdas H3 en cada llamada (no hay parches
// parciales — negocio siempre decide la tabla de precios completa junta).
type ZoneFareRuleRequest struct {
	Name                          string        `json:"name" validate:"required" example:"Charallave"`
	IncludedKm                    float64       `json:"included_km" validate:"required,gt=0" example:"3"`
	DayMinFareUSD                 float64       `json:"day_min_fare_usd" validate:"required,gt=0" example:"1.2375"`
	DayExtraRatePerKmUSD          float64       `json:"day_extra_rate_per_km_usd" validate:"gte=0" example:"0.30"`
	NightMinFareUSD               float64       `json:"night_min_fare_usd" validate:"required,gt=0" example:"1.575"`
	NightExtraRatePerKmUSD        float64       `json:"night_extra_rate_per_km_usd" validate:"gte=0" example:"0.50"`
	EarlyMorningMinFareUSD        float64       `json:"early_morning_min_fare_usd" validate:"required,gt=0" example:"1.80"`
	EarlyMorningExtraRatePerKmUSD float64       `json:"early_morning_extra_rate_per_km_usd" validate:"gte=0" example:"0.70"`
	Polygon                       []LatLngPoint `json:"polygon" validate:"required,min=3,dive"`
}

func (r *ZoneFareRuleRequest) Validate() error {
	return tripValidate.StructCtx(context.Background(), r)
}

type ZoneFareRuleResponse struct {
	ZoneID                        uuid.UUID `json:"zone_id"`
	Name                          string    `json:"name"`
	IncludedKm                    float64   `json:"included_km"`
	DayMinFareUSD                 float64   `json:"day_min_fare_usd"`
	DayExtraRatePerKmUSD          float64   `json:"day_extra_rate_per_km_usd"`
	NightMinFareUSD               float64   `json:"night_min_fare_usd"`
	NightExtraRatePerKmUSD        float64   `json:"night_extra_rate_per_km_usd"`
	EarlyMorningMinFareUSD        float64   `json:"early_morning_min_fare_usd"`
	EarlyMorningExtraRatePerKmUSD float64   `json:"early_morning_extra_rate_per_km_usd"`
	// H3Cells es el resultado de convertir el polígono enviado; el CRM lo usa
	// para redibujar la zona al editarla.
	H3Cells   []string  `json:"h3_cells"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ZoneFareRulesResponse struct {
	Zones []ZoneFareRuleResponse `json:"zones"`
}
