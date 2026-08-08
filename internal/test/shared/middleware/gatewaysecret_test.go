package middleware_test

import (
	"net/http"
	"testing"

	"Encargalo.app-api.go/internal/shared/buildhttptest"
	"Encargalo.app-api.go/internal/shared/config"
	sharedmw "Encargalo.app-api.go/internal/shared/middleware"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

const validGatewaySecret = "test-internal-gateway-secret"

func newGatewaySecretAuth() sharedmw.GatewaySecretAuth {
	return sharedmw.NewGatewaySecretAuth(config.Config{InternalGateway: config.InternalGateway{Secret: validGatewaySecret}})
}

func TestGatewaySecretAuth_ValidSecret(t *testing.T) {
	g := newGatewaySecretAuth()
	tc := buildhttptest.SetupControllerCase("/internal/verify-session", http.MethodPost, nil)
	tc.Req.Header.Set(sharedmw.GatewaySecretHeader, validGatewaySecret)
	tc.Ctx.SetRequest(tc.Req)

	err := g.Validate(nextOK)(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
}

func TestGatewaySecretAuth_MissingHeader(t *testing.T) {
	g := newGatewaySecretAuth()
	tc := buildhttptest.SetupControllerCase("/internal/verify-session", http.MethodPost, nil)

	err := g.Validate(nextOK)(tc.Ctx)

	he, ok := err.(*echo.HTTPError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusUnauthorized, he.Code)
}

func TestGatewaySecretAuth_WrongSecret(t *testing.T) {
	g := newGatewaySecretAuth()
	tc := buildhttptest.SetupControllerCase("/internal/verify-session", http.MethodPost, nil)
	tc.Req.Header.Set(sharedmw.GatewaySecretHeader, "wrong-secret")
	tc.Ctx.SetRequest(tc.Req)

	err := g.Validate(nextOK)(tc.Ctx)

	he, ok := err.(*echo.HTTPError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusUnauthorized, he.Code)
}
