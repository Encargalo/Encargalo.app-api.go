package repo

import (
	"context"
	"encoding/json"
	"time"

	"Encargalo.app-api.go/internal/auth/domain/models"
	"Encargalo.app-api.go/internal/auth/domain/ports"
	"Encargalo.app-api.go/internal/shared/applog"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type authRepo struct {
	redis *redis.Client
}

func NewAuthRepo(redis *redis.Client) ports.AuthRepo {
	return &authRepo{redis}
}

const sessionKeyPrefix = "sessions:"
const userSessionsKeyPrefix = "user_sessions:"
const sessionTTL = 365 * 24 * time.Hour

func sessionKey(id uuid.UUID) string { return sessionKeyPrefix + id.String() }

func userSessionsKey(userID uuid.UUID) string { return userSessionsKeyPrefix + userID.String() }

func (s *authRepo) SaveSession(ctx context.Context, session *models.ActiveSession) error {

	sessionData, _ := json.Marshal(session)

	if err := s.redis.Set(ctx, sessionKey(session.ID), sessionData, sessionTTL).Err(); err != nil {
		applog.Error(ctx, "error al registrar la sesion", err, "session_id", session.ID.String())
		return errcustom.ErrUnexpectedError
	}

	// Índice secundario por usuario: permite invalidar todas las sesiones de
	// una cuenta (todos los dispositivos) sin conocer cada session_id de
	// antemano — usado al solicitar la eliminación de la cuenta. Best-effort:
	// si falla, la sesión principal ya quedó guardada y el login no se rompe.
	pipe := s.redis.Pipeline()
	pipe.SAdd(ctx, userSessionsKey(session.UserID), session.ID.String())
	pipe.Expire(ctx, userSessionsKey(session.UserID), sessionTTL)
	if _, err := pipe.Exec(ctx); err != nil {
		applog.Error(ctx, "error al indexar la sesion por usuario", err, "user_id", session.UserID.String())
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
		return nil, errcustom.ErrUnexpectedError
	}

	var session models.ActiveSession
	if err := json.Unmarshal([]byte(sessionData), &session); err != nil {
		applog.Error(ctx, "error al deserializar la sesion", err, "session_id", sessionID.String())
		return nil, errcustom.ErrUnexpectedError
	}

	return &session, nil
}

func (s *authRepo) DeleteSession(ctx context.Context, session_id uuid.UUID) error {
	if err := s.redis.Del(ctx, sessionKey(session_id)).Err(); err != nil {
		applog.Error(ctx, "error al eliminar la sesion", err, "session_id", session_id.String())
		return errcustom.ErrUnexpectedError
	}

	return nil
}

// DeleteAllSessionsForUser invalida todas las sesiones activas de un usuario
// en todos los dispositivos (usado al solicitar la eliminación de cuenta, para
// que "volver a iniciar sesión" sea una señal inequívoca sin importar desde
// qué dispositivo se pidió el borrado). No-op seguro si no tiene sesiones.
func (s *authRepo) DeleteAllSessionsForUser(ctx context.Context, userID uuid.UUID) error {
	key := userSessionsKey(userID)

	ids, err := s.redis.SMembers(ctx, key).Result()
	if err != nil && err != redis.Nil {
		applog.Error(ctx, "error al listar sesiones del usuario", err, "user_id", userID.String())
		return errcustom.ErrUnexpectedError
	}

	if len(ids) > 0 {
		keys := make([]string, len(ids))
		for i, id := range ids {
			keys[i] = sessionKeyPrefix + id
		}
		if err := s.redis.Del(ctx, keys...).Err(); err != nil {
			applog.Error(ctx, "error al eliminar sesiones del usuario", err, "user_id", userID.String())
			return errcustom.ErrUnexpectedError
		}
	}

	if err := s.redis.Del(ctx, key).Err(); err != nil {
		applog.Error(ctx, "error al eliminar el indice de sesiones del usuario", err, "user_id", userID.String())
		return errcustom.ErrUnexpectedError
	}

	return nil
}
