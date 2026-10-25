package models

import (
	"time"

	"github.com/google/uuid"
)

// RiderPresence es lo que se publica de un rider conectado: su última ubicación
// más los atributos por los que el matching filtra.
type RiderPresence struct {
	RiderID  uuid.UUID
	Location RiderLocation
	Mode     RiderMode
	Vehicle  string
}

// PresenceQuery busca riders conectados alrededor de un punto. MaxAge descarta
// ubicaciones viejas; los umbrales difieren entre el mapa y el matching.
type PresenceQuery struct {
	Latitude  float64
	Longitude float64
	RadiusKm  float64
	Mode      RiderMode
	MaxAge    time.Duration
	// Limit acota el resultado ya ordenado por distancia; 0 es sin tope.
	Limit int
	// ExcludeRiders es un set porque el matching ya arma uno (offerExclusions).
	ExcludeRiders   map[uuid.UUID]struct{}
	ExcludeVehicles []string
}

// PresenceTTL expira al rider que dejó de reportar. Supera a TripMatchRiderMaxAge
// a propósito: la frescura la decide MaxAge, el TTL solo recoge la basura.
const PresenceTTL = 5 * time.Minute
