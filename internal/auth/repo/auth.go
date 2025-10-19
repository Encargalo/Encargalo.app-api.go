package repo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"Encargalo.app-api.go/internal/auth/domain/models"
	"Encargalo.app-api.go/internal/auth/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/google/uuid"
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

func (s *authRepo) SearchSession(ctx context.Context, sessionID uuid.UUID) (*models.ActiveSession, error) {

	if sessionID == uuid.Nil {
		return nil, errors.New("invalid session ID")
	}

	sessionIDStr := sessionID.String()

	sessionData, err := s.redis.Get(ctx, sessionIDStr).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, errors.New("session not found")
		}
		fmt.Println("Error getting session:", err)
		return nil, errcustom.ErrUnexpectedError
	}

	var session models.ActiveSession
	if err := json.Unmarshal([]byte(sessionData), &session); err != nil {
		fmt.Println("Error unmarshalling session data:", err)
		return nil, errcustom.ErrUnexpectedError
	}

	return &session, nil
}
