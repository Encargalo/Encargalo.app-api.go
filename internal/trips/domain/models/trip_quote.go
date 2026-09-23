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
	FareUSD          float64
	FareBs           float64
	BcvRate          float64
	CreatedAt        time.Time
	ExpiresAt        time.Time
}
