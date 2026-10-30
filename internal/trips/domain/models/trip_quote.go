package models

import (
	"time"

	"github.com/google/uuid"
)

// TripQuoteTTL es más corto que orders (30m) porque acá no hay que esperar un pago.
const TripQuoteTTL = 10 * time.Minute

// TripQuote congela trayecto y precio hasta que el customer confirme el viaje o expire.
type TripQuote struct {
	QuoteID          uuid.UUID
	CustomerID       uuid.UUID
	PickupAddress    string
	PickupLatitude   float64
	PickupLongitude  float64
	DropoffAddress   string
	DropoffLatitude  float64
	DropoffLongitude float64
	DistanceMeters   float64
	DurationMin      int64
	// FareUSD ya trae restado DiscountUSD y sumado PickupSurchargeUSD: es lo que paga el cliente.
	FareUSD            float64
	DiscountUSD        float64
	PickupSurchargeUSD float64
	FareBs             float64
	BcvRate            float64
	// FareBand es el tramo horario que produjo FareUSD, congelado junto con él.
	FareBand string
	// VehicleType es el servicio elegido; el viaje lo hereda de acá, no del request.
	VehicleType string
	CreatedAt   time.Time
	ExpiresAt   time.Time
}
