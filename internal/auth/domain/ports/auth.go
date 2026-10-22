package ports

import (
	"context"

	"Encargalo.app-api.go/internal/auth/domain/models"
	"github.com/google/uuid"
)

type AuthApp interface {
	SignInCustomer(ctx context.Context, phone, password string) (uuid.UUID, error)
	SignInEmployee(ctx context.Context, phone, password string) (uuid.UUID, error)
	SignInRider(ctx context.Context, phone, password string) (uuid.UUID, error)
	IdentifyRiderPhone(ctx context.Context, phone string) (models.RiderPhoneNext, error)
	SignInStaff(ctx context.Context, phone, password string) (uuid.UUID, error)
	SignInGoogle(ctx context.Context, idToken string) (uuid.UUID, error)
	SearchSessions(ctx context.Context, session_id uuid.UUID) (*models.ActiveSession, error)
	DeleteSession(ctx context.Context, session_id uuid.UUID) error
	DeleteAllSessionsForUser(ctx context.Context, userID uuid.UUID) error
	EnsureAccountEnabled(ctx context.Context, userID uuid.UUID) error
}

type AuthRepo interface {
	SaveSession(ctx context.Context, session *models.ActiveSession) error
	SearchSession(ctx context.Context, sessionID uuid.UUID) (*models.ActiveSession, error)
	DeleteSession(ctx context.Context, session_id uuid.UUID) error
	DeleteAllSessionsForUser(ctx context.Context, userID uuid.UUID) error
}
