package ports

import (
	"context"

	"Encargalo.app-api.go/internal/auth/domain/models"
	"github.com/google/uuid"
)

type AuthApp interface {
	SignInCustomer(ctx context.Context, phone, password string) (uuid.UUID, error)
}

type AuthRepo interface {
	SaveSession(ctx context.Context, session *models.ActiveSession) error
}
