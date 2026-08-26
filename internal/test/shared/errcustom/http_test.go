package errcustom_test

import (
	"errors"
	"net/http"
	"testing"

	"Encargalo.app-api.go/internal/shared/buildhttptest"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/shared/response"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestHTTPErrorHandler_V1_SentinelMapped(t *testing.T) {
	tc := buildhttptest.SetupControllerCase("/x", http.MethodGet, nil)

	errcustom.HTTPErrorHandler(errcustom.ErrPhoneAlreadyExist, tc.Ctx)

	assert.Equal(t, http.StatusConflict, tc.Res.Code)
	assert.JSONEq(t, `"el teléfono ya existe"`, tc.Res.Body.String())
}

func TestHTTPErrorHandler_V1_AdHocEchoHTTPError(t *testing.T) {
	tc := buildhttptest.SetupControllerCase("/x", http.MethodGet, nil)

	err := echo.NewHTTPError(http.StatusBadRequest, "shop_id es requerido")
	errcustom.HTTPErrorHandler(err, tc.Ctx)

	assert.Equal(t, http.StatusBadRequest, tc.Res.Code)
	assert.JSONEq(t, `"shop_id es requerido"`, tc.Res.Body.String())
}

func TestHTTPErrorHandler_V1_UnknownError(t *testing.T) {
	tc := buildhttptest.SetupControllerCase("/x", http.MethodGet, nil)

	errcustom.HTTPErrorHandler(errors.New("boom"), tc.Ctx)

	assert.Equal(t, http.StatusInternalServerError, tc.Res.Code)
	assert.JSONEq(t, `"error inesperado"`, tc.Res.Body.String())
}

func TestHTTPErrorHandler_V2_SentinelMapped(t *testing.T) {
	tc := buildhttptest.SetupControllerCase("/x", http.MethodGet, nil)
	tc.Req.Header.Set(response.VersionHeader, response.VersionV2)

	errcustom.HTTPErrorHandler(errcustom.ErrPhoneAlreadyExist, tc.Ctx)

	assert.Equal(t, http.StatusConflict, tc.Res.Code)
	assert.JSONEq(t, `{
		"data": null,
		"error": {"code": "phone_already_exist", "message": "el teléfono ya existe"}
	}`, tc.Res.Body.String())
}

func TestHTTPErrorHandler_V2_AdHocEchoHTTPError(t *testing.T) {
	tc := buildhttptest.SetupControllerCase("/x", http.MethodGet, nil)
	tc.Req.Header.Set(response.VersionHeader, response.VersionV2)

	err := echo.NewHTTPError(http.StatusBadRequest, "shop_id es requerido")
	errcustom.HTTPErrorHandler(err, tc.Ctx)

	assert.Equal(t, http.StatusBadRequest, tc.Res.Code)
	assert.JSONEq(t, `{
		"data": null,
		"error": {"code": "bad_request", "message": "shop_id es requerido"}
	}`, tc.Res.Body.String())
}

func TestHTTPErrorHandler_V2_UnknownError(t *testing.T) {
	tc := buildhttptest.SetupControllerCase("/x", http.MethodGet, nil)
	tc.Req.Header.Set(response.VersionHeader, response.VersionV2)

	errcustom.HTTPErrorHandler(errors.New("boom"), tc.Ctx)

	assert.Equal(t, http.StatusInternalServerError, tc.Res.Code)
	assert.JSONEq(t, `{
		"data": null,
		"error": {"code": "unexpected_error", "message": "error inesperado"}
	}`, tc.Res.Body.String())
}

func TestHTTPErrorHandler_AlreadyCommitted_NoOp(t *testing.T) {
	tc := buildhttptest.SetupControllerCase("/x", http.MethodGet, nil)
	assert.NoError(t, tc.Ctx.NoContent(http.StatusTeapot))

	errcustom.HTTPErrorHandler(errcustom.ErrNotFound, tc.Ctx)

	assert.Equal(t, http.StatusTeapot, tc.Res.Code)
}

func TestCode_KnownSentinel(t *testing.T) {
	assert.Equal(t, "phone_already_exist", errcustom.Code(errcustom.ErrPhoneAlreadyExist))
	assert.Equal(t, "shop_onboarding_incomplete", errcustom.Code(errcustom.ErrShopOnboardingIncomplete))
	assert.Equal(t, "rider_profile_incomplete", errcustom.Code(errcustom.ErrRiderProfileIncomplete))
}

func TestHTTPStatus(t *testing.T) {
	assert.Equal(t, http.StatusConflict, errcustom.HTTPStatus(errcustom.ErrPhoneAlreadyExist))
	assert.Equal(t, http.StatusInternalServerError, errcustom.HTTPStatus(errors.New("boom")))
}
