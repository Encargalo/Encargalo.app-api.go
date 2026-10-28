package ports

import (
	"context"

	"Encargalo.app-api.go/internal/riders/domain/models"
)

// LastKnownLocationStore persiste la última ubicación reportada por cada
// rider sin TTL (a diferencia de RiderPresenceStore, que vive en Redis con
// 45s de vida). Alimenta el fallback de matching para riders sin presencia
// viva, acotado por models.LastKnownLocationMaxAge.
type LastKnownLocationStore interface {
	Save(ctx context.Context, loc models.RiderLastKnownLocation) error
	SearchNearby(ctx context.Context, query models.LastKnownLocationQuery) ([]models.RiderLastKnownLocation, error)
}
