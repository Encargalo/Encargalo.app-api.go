package auth

import (
	"context"
	"net/http"
	"time"

	"Encargalo.app-api.go/internal/auth/domain/models"
	"Encargalo.app-api.go/internal/auth/domain/ports"
	"Encargalo.app-api.go/internal/shared/jwt"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type AuthMiddleware interface {
	Auth(next echo.HandlerFunc) echo.HandlerFunc
	Require(allowed ...models.Role) echo.MiddlewareFunc
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

		if cookie, err := c.Cookie("encargalo-session"); err == nil {
			tokenString = cookie.Value
		}

		claims, err := a.jwt.ValidateToken(tokenString)
		if err != nil {
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid token")
		}

		rawID, ok := claims["session_id"].(string)
		if !ok {
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid session_id")
		}

		sessionID, err := uuid.Parse(rawID)
		if err != nil {
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid session_id")
		}

		session, err := a.svc.SearchSessions(ctx, sessionID)
		if err != nil {
			return echo.NewHTTPError(http.StatusUnauthorized, "session not found")
		}

		if time.Now().After(session.ExpiresAt) {
			return echo.NewHTTPError(http.StatusUnauthorized, "session expired")
		}

		ctx = context.WithValue(ctx, models.SessionKey, *session)
		c.SetRequest(req.WithContext(ctx))

		return next(c)
	}
}

func (a *auth) Require(allowed ...models.Role) echo.MiddlewareFunc {
	allowedMap := make(map[models.Role]struct{}, len(allowed))
	for _, t := range allowed {
		allowedMap[t] = struct{}{}
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			session, ok := c.Request().Context().Value(models.SessionKey).(models.ActiveSession)
			if !ok {
				return echo.NewHTTPError(http.StatusUnauthorized, "session not found in context")
			}

			for _, role := range session.Roles {
				if _, ok := allowedMap[role]; ok {
					return next(c)
				}
			}

			return echo.NewHTTPError(http.StatusForbidden, "access denied")
		}
	}
}
