package ports

import (
	"context"

	"Encargalo.app-api.go/internal/users/domain/dto"
	"github.com/google/uuid"
)

type CustomersSessionsApp interface {
	Sign_In(ctx context.Context, sign_in dto.SignIn) (uuid.UUID, error)
}
