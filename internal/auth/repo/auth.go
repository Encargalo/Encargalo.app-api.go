package repo

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"Encargalo.app-api.go/internal/auth/domain/models"
	"Encargalo.app-api.go/internal/auth/domain/ports"
	"Encargalo.app-api.go/internal/pkg/logs"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type authRepo struct {
	redis     *redis.Client
	slackLogs logs.Logs
}

func NewAuthRepo(redis *redis.Client, slackLogs logs.Logs) ports.AuthRepo {
	return &authRepo{redis, slackLogs}
}

func (s *authRepo) SaveSession(ctx context.Context, session *models.ActiveSession) error {

	sessionData, _ := json.Marshal(session)

	if err := s.redis.Set(ctx, session.ID.String(), sessionData, 365*24*time.Hour).Err(); err != nil {
		slog.Error("error al registrar la sesion - "+session.ID.String(), "error", err)
		s.slackLogs.Slack(err)
		return errcustom.ErrUnexpectedError
	}

	return nil
}

func (s *authRepo) SearchSession(ctx context.Context, sessionID uuid.UUID) (*models.ActiveSession, error) {

	sessionIDStr := sessionID.String()

	sessionData, err := s.redis.Get(ctx, sessionIDStr).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, errcustom.ErrNotFound
		}
		slog.Error("Error getting session:", "error", err)
		s.slackLogs.Slack(err)
		return nil, errcustom.ErrUnexpectedError
	}

	var session models.ActiveSession
	if err := json.Unmarshal([]byte(sessionData), &session); err != nil {
		slog.Error("Error unmarshalling session data:", "error", err)
		s.slackLogs.Slack(err)
		return nil, errcustom.ErrUnexpectedError
	}

	return &session, nil
}

func (s *authRepo) DeleteSession(ctx context.Context, session_id uuid.UUID) error {
	sessionIDStr := session_id.String()

	if err := s.redis.Del(ctx, sessionIDStr).Err(); err != nil {
		slog.Error("Error deleting session:", "error", err)
		s.slackLogs.Slack(err)
		return errcustom.ErrUnexpectedError
	}

	return nil
}
