package middleware

import (
	"crypto/subtle"
	"net/http"

	"Encargalo.app-api.go/internal/shared/config"
	"github.com/labstack/echo/v4"
)

// GatewaySecretHeader es el header que el reverse proxy (Traefik) inyecta en
// cada request que reenvía al backend, con el secreto compartido configurado
// en ambos lados.
const GatewaySecretHeader = "X-Internal-Gateway-Secret"

// SessionHeader lleva el JSON de models.ActiveSession que produce
// /internal/verify-session y que el reverse proxy copia hacia la request que
// reenvía al backend real (ver ForwardAuth authResponseHeaders).
const SessionHeader = "X-Encargalo-Session"

// GatewaySecretAuth valida que la request venga del reverse proxy y no
// directo de internet, como defensa en profundidad además del aislamiento de
// red privada.
type GatewaySecretAuth interface {
	Validate(next echo.HandlerFunc) echo.HandlerFunc
}

type gatewaySecretAuth struct {
	secret string
}

func NewGatewaySecretAuth(cfg config.Config) GatewaySecretAuth {
	return &gatewaySecretAuth{secret: cfg.InternalGateway.Secret}
}

func (g *gatewaySecretAuth) Validate(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		token := c.Request().Header.Get(GatewaySecretHeader)
		if subtle.ConstantTimeCompare([]byte(token), []byte(g.secret)) != 1 {
			return echo.NewHTTPError(http.StatusUnauthorized, "no autorizado")
		}

		return next(c)
	}
}
