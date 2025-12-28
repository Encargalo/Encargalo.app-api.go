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

type SessionType string

const (
	SessionTypeCustomer SessionType = "customer-session"
	SessionTypeBusiness SessionType = "employee-session"
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

		var tokenString string

		// Intentamos buscar primero la cookie de cliente
		// if cookie, err := c.Cookie(string(SessionTypeCustomer)); err == nil {
		// 	tokenString = cookie.Value
		// } else if cookie, err := c.Cookie(string(SessionTypeBusiness)); err == nil {
		// 	// Si no está la de cliente, buscamos la de negocio
		// 	tokenString = cookie.Value
		// } else {
		// 	// Si no está ninguna de las dos
		// 	return echo.NewHTTPError(http.StatusUnauthorized, "missing or invalid session cookie")
		// }

		// 1. Buscamos el token en la cabecera Authorization
		if cookie, err := c.Cookie("encargalo-session"); err == nil {
			tokenString = cookie.Value
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

		ctx = context.WithValue(ctx, "session", *session)

		c.SetRequest(req.WithContext(ctx))

		return next(c)
	}
}
