package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// LastKnownLocationMaxAge es el tope de antigüedad para usar la última
// ubicación conocida como respaldo de matching cuando no hay presencia viva
// en Redis. Sin tracking en background, es literalmente la última posición
// reportada en la última sesión con la app abierta — más vieja que esto, el
// rider probablemente ya se movió de zona.
const LastKnownLocationMaxAge = 12 * time.Hour

// RiderLastKnownLocation persiste, sin TTL, lo último que RiderPresenceStore
// también guarda en Redis con TTL de 45s. Alimenta el fallback de matching
// para pushear a riders que cerraron la app (sin presencia viva).
type RiderLastKnownLocation struct {
	bun.BaseModel `bun:"table:riders.last_known_locations"`

	RiderID   uuid.UUID `bun:"rider_id,pk,type:uuid"`
	Latitude  float64   `bun:"latitude,notnull"`
	Longitude float64   `bun:"longitude,notnull"`
	Mode      RiderMode `bun:"mode,notnull"`
	// Vehicle es canónico (ver NormalizeVehicleType), igual que RiderPresence.Vehicle.
	Vehicle   string    `bun:"vehicle"`
	UpdatedAt time.Time `bun:"updated_at,default:now()"`
}

// LastKnownLocationQuery busca riders por última ubicación conocida dentro
// de un radio, acotado por antigüedad máxima y modo. Misma forma que
// PresenceQuery, para que el fallback filtre igual que el matching en vivo.
type LastKnownLocationQuery struct {
	Latitude        float64
	Longitude       float64
	RadiusKm        float64
	Mode            RiderMode
	MaxAge          time.Duration
	ExcludeVehicles []string
	// Exclude descarta riderIDs que ya están en el pool en vivo o ya fueron ofertados.
	Exclude map[uuid.UUID]struct{}
}
