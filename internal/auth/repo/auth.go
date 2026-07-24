package repo

import (
	"context"
	"encoding/json"
	"time"

	"Encargalo.app-api.go/internal/auth/domain/models"
	"Encargalo.app-api.go/internal/auth/domain/ports"
	"Encargalo.app-api.go/internal/pkg/logs"
	"Encargalo.app-api.go/internal/shared/applog"
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

const sessionKeyPrefix = "sessions:"

func sessionKey(id uuid.UUID) string { return sessionKeyPrefix + id.String() }

func (s *authRepo) SaveSession(ctx context.Context, session *models.ActiveSession) error {

	sessionData, _ := json.Marshal(session)

	if err := s.redis.Set(ctx, sessionKey(session.ID), sessionData, 365*24*time.Hour).Err(); err != nil {
		applog.Error(ctx, "error al registrar la sesion", err, "session_id", session.ID.String())
		s.slackLogs.Slack(err)
		return errcustom.ErrUnexpectedError
	}

	return nil
}

func (s *authRepo) SearchSession(ctx context.Context, sessionID uuid.UUID) (*models.ActiveSession, error) {

	sessionData, err := s.redis.Get(ctx, sessionKey(sessionID)).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, errcustom.ErrNotFound
		}
		applog.Error(ctx, "error al obtener la sesion", err, "session_id", sessionID.String())
		s.slackLogs.Slack(err)
		return nil, errcustom.ErrUnexpectedError
	}

	var session models.ActiveSession
	if err := json.Unmarshal([]byte(sessionData), &session); err != nil {
		applog.Error(ctx, "error al deserializar la sesion", err, "session_id", sessionID.String())
		s.slackLogs.Slack(err)
		return nil, errcustom.ErrUnexpectedError
	}

	return &session, nil
}

func (s *authRepo) DeleteSession(ctx context.Context, session_id uuid.UUID) error {
	if err := s.redis.Del(ctx, sessionKey(session_id)).Err(); err != nil {
		applog.Error(ctx, "error al eliminar la sesion", err, "session_id", session_id.String())
		s.slackLogs.Slack(err)
		return errcustom.ErrUnexpectedError
	}

	return nil
}
