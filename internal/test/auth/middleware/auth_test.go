package middleware_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	authmw "Encargalo.app-api.go/internal/auth/domain/middleware/auth"
	"Encargalo.app-api.go/internal/auth/domain/models"
	"Encargalo.app-api.go/internal/shared/buildhttptest"
	"Encargalo.app-api.go/internal/shared/config"
	sharedmw "Encargalo.app-api.go/internal/shared/middleware"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

const validSecret = "test-internal-gateway-secret"

func newAuthMiddleware() authmw.AuthMiddleware {
	gatewaySecret := sharedmw.NewGatewaySecretAuth(config.Config{InternalGateway: config.InternalGateway{Secret: validSecret}})
	return authmw.NewAuthMidlleware(gatewaySecret)
}

func nextOK(c echo.Context) error {
	return c.NoContent(http.StatusOK)
}

func TestAuth_ValidSessionHeader_SetsContextAndCallsNext(t *testing.T) {
	a := newAuthMiddleware()
	session := models.ActiveSession{
		ID:     uuid.New(),
		UserID: uuid.New(),
		Roles:  []models.Role{models.Customer},
	}
	raw, err := json.Marshal(session)
	assert.NoError(t, err)

	tc := buildhttptest.SetupControllerCase("/business/products", http.MethodGet, nil)
	tc.Req.Header.Set(sharedmw.GatewaySecretHeader, validSecret)
	tc.Req.Header.Set(sharedmw.SessionHeader, string(raw))
	tc.Ctx.SetRequest(tc.Req)

	var gotSession models.ActiveSession
	next := func(c echo.Context) error {
		gotSession = c.Request().Context().Value(models.SessionKey).(models.ActiveSession)
		return c.NoContent(http.StatusOK)
	}

	err = a.Auth(next)(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
	assert.Equal(t, session.ID, gotSession.ID)
	assert.Equal(t, session.Roles, gotSession.Roles)
}

func TestAuth_MissingSessionHeader_Unauthorized(t *testing.T) {
	a := newAuthMiddleware()
	tc := buildhttptest.SetupControllerCase("/business/products", http.MethodGet, nil)
	tc.Req.Header.Set(sharedmw.GatewaySecretHeader, validSecret)
	tc.Ctx.SetRequest(tc.Req)

	err := a.Auth(nextOK)(tc.Ctx)

	he, ok := err.(*echo.HTTPError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusUnauthorized, he.Code)
}

func TestAuth_InvalidJSONSessionHeader_Unauthorized(t *testing.T) {
	a := newAuthMiddleware()
	tc := buildhttptest.SetupControllerCase("/business/products", http.MethodGet, nil)
	tc.Req.Header.Set(sharedmw.GatewaySecretHeader, validSecret)
	tc.Req.Header.Set(sharedmw.SessionHeader, "{not-json")
	tc.Ctx.SetRequest(tc.Req)

	err := a.Auth(nextOK)(tc.Ctx)

	he, ok := err.(*echo.HTTPError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusUnauthorized, he.Code)
}

func TestAuth_MissingGatewaySecret_Unauthorized(t *testing.T) {
	a := newAuthMiddleware()
	session := models.ActiveSession{ID: uuid.New()}
	raw, _ := json.Marshal(session)

	tc := buildhttptest.SetupControllerCase("/business/products", http.MethodGet, nil)
	tc.Req.Header.Set(sharedmw.SessionHeader, string(raw))
	tc.Ctx.SetRequest(tc.Req)

	err := a.Auth(nextOK)(tc.Ctx)

	he, ok := err.(*echo.HTTPError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusUnauthorized, he.Code)
}

func TestAuth_WrongGatewaySecret_Unauthorized(t *testing.T) {
	a := newAuthMiddleware()
	session := models.ActiveSession{ID: uuid.New()}
	raw, _ := json.Marshal(session)

	tc := buildhttptest.SetupControllerCase("/business/products", http.MethodGet, nil)
	tc.Req.Header.Set(sharedmw.GatewaySecretHeader, "wrong-secret")
	tc.Req.Header.Set(sharedmw.SessionHeader, string(raw))
	tc.Ctx.SetRequest(tc.Req)

	err := a.Auth(nextOK)(tc.Ctx)

	he, ok := err.(*echo.HTTPError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusUnauthorized, he.Code)
}

func TestRequire_AllowedRole_CallsNext(t *testing.T) {
	a := newAuthMiddleware()
	session := models.ActiveSession{Roles: []models.Role{models.Owner}}

	tc := buildhttptest.SetupControllerCase("/business/shops", http.MethodGet, nil)
	ctx := context.WithValue(tc.Req.Context(), models.SessionKey, session)
	tc.Ctx.SetRequest(tc.Req.WithContext(ctx))

	err := a.Require(models.OrderManager, models.Owner)(nextOK)(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
}

func TestRequire_DisallowedRole_Forbidden(t *testing.T) {
	a := newAuthMiddleware()
	session := models.ActiveSession{Roles: []models.Role{models.Customer}}

	tc := buildhttptest.SetupControllerCase("/business/shops", http.MethodGet, nil)
	ctx := context.WithValue(tc.Req.Context(), models.SessionKey, session)
	tc.Ctx.SetRequest(tc.Req.WithContext(ctx))

	err := a.Require(models.OrderManager, models.Owner)(nextOK)(tc.Ctx)

	he, ok := err.(*echo.HTTPError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusForbidden, he.Code)
}

func TestRequire_NoSessionInContext_Unauthorized(t *testing.T) {
	a := newAuthMiddleware()
	tc := buildhttptest.SetupControllerCase("/business/shops", http.MethodGet, nil)

	err := a.Require(models.Owner)(nextOK)(tc.Ctx)

	he, ok := err.(*echo.HTTPError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusUnauthorized, he.Code)
}
