package ports

import (
	"context"

	"Encargalo.app-api.go/internal/tracking/domain/dtos"
	"Encargalo.app-api.go/internal/tracking/domain/models"
	"github.com/google/uuid"
)

type TrackingApp interface {
	Track(ctx context.Context, userID, sessionID uuid.UUID, event dtos.TrackEvent) error
}

type TrackingRepo interface {
	Insert(ctx context.Context, event models.Event) error
}
