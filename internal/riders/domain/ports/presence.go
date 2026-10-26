package ports

import (
	"context"

	"Encargalo.app-api.go/internal/riders/domain/models"
	"github.com/google/uuid"
)

// RiderPresenceStore guarda la ubicación en vivo de los riders conectados fuera
// del proceso, para que cualquier réplica pueda responder quién está cerca.
type RiderPresenceStore interface {
	// SaveLocation publica la presencia del rider y la saca del índice del otro
	// modo: un rider que pasa de pedidos a viajes no puede quedar en los dos.
	SaveLocation(ctx context.Context, presence models.RiderPresence) error
	// Remove borra al rider de los dos índices y su presencia.
	Remove(ctx context.Context, riderID uuid.UUID) error
	// SearchNearby devuelve los riders del modo pedido dentro del radio,
	// ordenados por distancia al punto.
	SearchNearby(ctx context.Context, query models.PresenceQuery) ([]models.RiderLocationEntry, error)
}
