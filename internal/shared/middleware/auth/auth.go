package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"Encargalo.app-api.go/internal/auth/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/shared/jwt"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// 1. Definimos el "Enum" para los tipos de sesión
type SessionType string

const (
	SessionTypeCustomer SessionType = "customer-session"
	SessionTypeBusiness SessionType = "business-session"
)

// Definimos keys para el contexto para evitar colisiones de strings
type contextKey string

const (
	ContextKeySession     contextKey = "session"
	ContextKeySessionType contextKey = "session_type"
)

type AuthMiddleware interface {
	Auth(next echo.HandlerFunc) echo.HandlerFunc
}

type auth struct {
	jwt jwt.Sessions
	svc ports.AuthApp
}

func NewAuthMidlleware(jwt jwt.Sessions, svc ports.AuthApp) AuthMiddleware {
	return &auth{jwt: jwt, svc: svc}
}

func (a *auth) Auth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		req := c.Request()
		ctx := req.Context()

		// 2. Lógica de detección de Cookie
		var tokenString string
		var currentSessionType SessionType

		// Intentamos buscar primero la cookie de cliente
		if cookie, err := c.Cookie(string(SessionTypeCustomer)); err == nil {
			tokenString = cookie.Value
			currentSessionType = SessionTypeCustomer
		} else if cookie, err := c.Cookie(string(SessionTypeBusiness)); err == nil {
			// Si no está la de cliente, buscamos la de negocio
			tokenString = cookie.Value
			currentSessionType = SessionTypeBusiness
		} else {
			// Si no está ninguna de las dos
			return echo.NewHTTPError(http.StatusUnauthorized, "missing or invalid session cookie")
		}

		// 3. Validamos el token obtenido (sea cual sea el origen)
		claims, err := a.jwt.ValidateToken(tokenString)
		if err != nil {
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid or expired token")
		}

		rawID, ok := claims["session_id"].(string)
		if !ok || strings.TrimSpace(rawID) == "" {
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid session_id in token")
		}

		sessionID, err := uuid.Parse(strings.TrimSpace(rawID))
		if err != nil {
			return echo.NewHTTPError(http.StatusUnauthorized, "malformed session_id")
		}

		session, err := a.svc.SearchSessions(ctx, sessionID)
		if err != nil {
			if errors.Is(err, errcustom.ErrNotFound) {
				return echo.NewHTTPError(http.StatusUnauthorized, "session not found")
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "error verifying session")
		}

		if time.Now().After(session.ExpiresAt) {
			return echo.NewHTTPError(http.StatusUnauthorized, "session expired")
		}

		// 4. Guardamos en el contexto LA SESIÓN y EL TIPO DE SESIÓN
		// Usamos tipos personalizados para las keys por buenas prácticas
		ctx = context.WithValue(ctx, "session", *session)
		ctx = context.WithValue(ctx, ContextKeySessionType, currentSessionType)

		c.SetRequest(req.WithContext(ctx))

		return next(c)
	}
}
