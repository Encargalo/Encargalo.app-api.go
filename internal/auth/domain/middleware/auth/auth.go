package auth

import (
	"context"
	"encoding/json"
	"net/http"

	"Encargalo.app-api.go/internal/auth/domain/models"
	sharedmw "Encargalo.app-api.go/internal/shared/middleware"
	"github.com/labstack/echo/v4"
)

type AuthMiddleware interface {
	Auth(next echo.HandlerFunc) echo.HandlerFunc
	Require(allowed ...models.Role) echo.MiddlewareFunc
}

type auth struct {
	gatewaySecret sharedmw.GatewaySecretAuth
}

func NewAuthMidlleware(gatewaySecret sharedmw.GatewaySecretAuth) AuthMiddleware {
	return &auth{gatewaySecret: gatewaySecret}
}

// Auth confía en la sesión que ya validó el reverse proxy (Traefik) vía su
// ForwardAuth contra /internal/verify-session — no decodifica JWT ni
// consulta la DB, solo lee el header X-Encargalo-Session. gatewaySecret.Validate
// rechaza cualquier request que no traiga el secreto compartido, como
// defensa en profundidad además del aislamiento de red privada.
func (a *auth) Auth(next echo.HandlerFunc) echo.HandlerFunc {
	return a.gatewaySecret.Validate(func(c echo.Context) error {
		req := c.Request()

		raw := req.Header.Get(sharedmw.SessionHeader)
		if raw == "" {
			return echo.NewHTTPError(http.StatusUnauthorized, "sesión no encontrada")
		}

		var session models.ActiveSession
		if err := json.Unmarshal([]byte(raw), &session); err != nil {
			return echo.NewHTTPError(http.StatusUnauthorized, "sesión inválida")
		}

		ctx := context.WithValue(req.Context(), models.SessionKey, session)
		c.SetRequest(req.WithContext(ctx))

		return next(c)
	})
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
				return echo.NewHTTPError(http.StatusUnauthorized, "sesión no encontrada en el contexto")
			}

			for _, role := range session.Roles {
				if _, ok := allowedMap[role]; ok {
					return next(c)
				}
			}

			return echo.NewHTTPError(http.StatusForbidden, "acceso denegado")
		}
	}
}
