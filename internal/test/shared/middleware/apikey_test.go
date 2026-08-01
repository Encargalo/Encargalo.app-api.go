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

const validKey = "test-external-api-key"

func newApiKeyAuth() sharedmw.ApiKeyAuth {
	return sharedmw.NewApiKeyAuth(config.Config{ExternalAPI: config.ExternalAPI{Key: validKey}})
}

func nextOK(c echo.Context) error {
	return c.NoContent(http.StatusOK)
}

func TestApiKeyAuth_ValidToken(t *testing.T) {
	a := newApiKeyAuth()
	tc := buildhttptest.SetupControllerCase("/bcv/rate", http.MethodGet, nil)
	tc.Req.Header.Set(echo.HeaderAuthorization, "Bearer "+validKey)
	tc.Ctx.SetRequest(tc.Req)

	err := a.Validate(nextOK)(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
}

func TestApiKeyAuth_MissingHeader(t *testing.T) {
	a := newApiKeyAuth()
	tc := buildhttptest.SetupControllerCase("/bcv/rate", http.MethodGet, nil)

	err := a.Validate(nextOK)(tc.Ctx)

	he, ok := err.(*echo.HTTPError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusUnauthorized, he.Code)
}

func TestApiKeyAuth_WrongScheme(t *testing.T) {
	a := newApiKeyAuth()
	tc := buildhttptest.SetupControllerCase("/bcv/rate", http.MethodGet, nil)
	tc.Req.Header.Set(echo.HeaderAuthorization, validKey)
	tc.Ctx.SetRequest(tc.Req)

	err := a.Validate(nextOK)(tc.Ctx)

	he, ok := err.(*echo.HTTPError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusUnauthorized, he.Code)
}

func TestApiKeyAuth_WrongKey(t *testing.T) {
	a := newApiKeyAuth()
	tc := buildhttptest.SetupControllerCase("/bcv/rate", http.MethodGet, nil)
	tc.Req.Header.Set(echo.HeaderAuthorization, "Bearer wrong-key")
	tc.Ctx.SetRequest(tc.Req)

	err := a.Validate(nextOK)(tc.Ctx)

	he, ok := err.(*echo.HTTPError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusUnauthorized, he.Code)
}
