package customerauth

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

type ctxKey string

const (
	ctxKeyCustomerID ctxKey = "customer_id"
	ctxKeySessionID  ctxKey = "session_id"
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

		cookie, err := c.Cookie("encargalo_session")
		if err != nil {
			return echo.NewHTTPError(http.StatusUnauthorized, "missing or invalid session cookie")
		}

		claims, err := a.jwt.ValidateToken(cookie.Value)
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

		ctx = context.WithValue(ctx, "customer_id", session.UserID)
		ctx = context.WithValue(ctx, "session_id", sessionID)

		c.SetRequest(req.WithContext(ctx))

		return next(c)
	}
}
