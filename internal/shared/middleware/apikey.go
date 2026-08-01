package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"Encargalo.app-api.go/internal/shared/config"
	"github.com/labstack/echo/v4"
)

const bearerPrefix = "Bearer "

// ApiKeyAuth valida clientes externos (no ligados a la sesión de usuario)
// mediante un API key estático enviado como "Authorization: Bearer <key>".
type ApiKeyAuth interface {
	Validate(next echo.HandlerFunc) echo.HandlerFunc
}

type apiKeyAuth struct {
	key string
}

func NewApiKeyAuth(cfg config.Config) ApiKeyAuth {
	return &apiKeyAuth{key: cfg.ExternalAPI.Key}
}

func (a *apiKeyAuth) Validate(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		header := c.Request().Header.Get(echo.HeaderAuthorization)
		if !strings.HasPrefix(header, bearerPrefix) {
			return echo.NewHTTPError(http.StatusUnauthorized, "token inválido")
		}

		token := strings.TrimPrefix(header, bearerPrefix)
		if subtle.ConstantTimeCompare([]byte(token), []byte(a.key)) != 1 {
			return echo.NewHTTPError(http.StatusUnauthorized, "token inválido")
		}

		return next(c)
	}
}
