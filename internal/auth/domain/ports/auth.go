package ports

import (
	"context"

	"Encargalo.app-api.go/internal/auth/domain/models"
	"github.com/google/uuid"
)

type AuthApp interface {
	SignInCustomer(ctx context.Context, phone, password string) (uuid.UUID, error)
	SearchSessions(ctx context.Context, session_id uuid.UUID) (*models.ActiveSession, error)
}

type AuthRepo interface {
	SaveSession(ctx context.Context, session *models.ActiveSession) error
	SearchSession(ctx context.Context, sessionID uuid.UUID) (*models.ActiveSession, error)
}
