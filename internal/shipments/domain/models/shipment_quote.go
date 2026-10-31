package models

import (
	"time"

	"github.com/google/uuid"
)

// ShipmentQuoteTTL, igual que trips: no hay que esperar un pago para confirmarla.
const ShipmentQuoteTTL = 10 * time.Minute

// ShipmentQuote congela trayecto y precio hasta que el customer confirme el envío o expire.
type ShipmentQuote struct {
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
	// FareUSD es lo que paga el cliente: misma fórmula que trips (tripfare), sin descuentos ni recargos propios.
	FareUSD   float64
	FareBs    float64
	BcvRate   float64
	FareBand  string
	CreatedAt time.Time
	ExpiresAt time.Time
}
