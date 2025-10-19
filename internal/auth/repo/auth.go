package repo

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"Encargalo.app-api.go/internal/auth/domain/models"
	"Encargalo.app-api.go/internal/auth/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/redis/go-redis/v9"
)

type authRepo struct {
	redis *redis.Client
}

func NewAuthRepo(redis *redis.Client) ports.AuthRepo {
	return &authRepo{redis: redis}
}

func (s *authRepo) SaveSession(ctx context.Context, session *models.ActiveSession) error {

	sessionData, _ := json.Marshal(session)

	if err := s.redis.Set(ctx, session.ID.String(), sessionData, 365*24*time.Hour).Err(); err != nil {
		fmt.Println("Error saving session:", err)
		return errcustom.ErrUnexpectedError
	}

	return nil
}
