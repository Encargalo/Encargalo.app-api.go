package handler_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"Encargalo.app-api.go/internal/auth/domain/models"
	authhandler "Encargalo.app-api.go/internal/auth/handler"
	"Encargalo.app-api.go/internal/auth/handler/request"
	mocksports "Encargalo.app-api.go/internal/mocks/auth/domain/ports"
	mockscookie "Encargalo.app-api.go/internal/mocks/pkg/cookie"
	mocksjwt "Encargalo.app-api.go/internal/mocks/shared/jwt"
	"Encargalo.app-api.go/internal/shared/buildhttptest"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIdentifyRiderPhone_SignIn(t *testing.T) {
	svc := mocksports.NewAuthApp(t)
	h := authhandler.NewAuthHandler(svc, mocksjwt.NewSessions(t), mockscookie.NewCookieHandler(t))

	body := buildBody(t, map[string]string{"phone_number": "+573001112233"})
	tc := buildhttptest.SetupControllerCase("/auth/riders/identify", http.MethodPost, body)
	svc.On("IdentifyRiderPhone", tc.Req.Context(), "+573001112233").Return(models.RiderPhoneNextSignIn, nil)

	err := h.IdentifyRiderPhone(tc.Ctx)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)

	var got request.IdentifyRiderPhoneResponse
	require.NoError(t, json.Unmarshal(tc.Res.Body.Bytes(), &got))
	assert.Equal(t, "sign_in", got.Next)
}

func TestIdentifyRiderPhone_Onboarding(t *testing.T) {
	svc := mocksports.NewAuthApp(t)
	h := authhandler.NewAuthHandler(svc, mocksjwt.NewSessions(t), mockscookie.NewCookieHandler(t))

	body := buildBody(t, map[string]string{"phone_number": "+573001112233"})
	tc := buildhttptest.SetupControllerCase("/auth/riders/identify", http.MethodPost, body)
	svc.On("IdentifyRiderPhone", tc.Req.Context(), "+573001112233").Return(models.RiderPhoneNextOnboarding, nil)

	err := h.IdentifyRiderPhone(tc.Ctx)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)

	var got request.IdentifyRiderPhoneResponse
	require.NoError(t, json.Unmarshal(tc.Res.Body.Bytes(), &got))
	assert.Equal(t, "onboarding", got.Next)
}

func TestIdentifyRiderPhone_Register(t *testing.T) {
	svc := mocksports.NewAuthApp(t)
	h := authhandler.NewAuthHandler(svc, mocksjwt.NewSessions(t), mockscookie.NewCookieHandler(t))

	body := buildBody(t, map[string]string{"phone_number": "+573001112233"})
	tc := buildhttptest.SetupControllerCase("/auth/riders/identify", http.MethodPost, body)
	svc.On("IdentifyRiderPhone", tc.Req.Context(), "+573001112233").Return(models.RiderPhoneNextRegister, nil)

	err := h.IdentifyRiderPhone(tc.Ctx)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)

	var got request.IdentifyRiderPhoneResponse
	require.NoError(t, json.Unmarshal(tc.Res.Body.Bytes(), &got))
	assert.Equal(t, "register", got.Next)
}

func TestIdentifyRiderPhone_InvalidPhone(t *testing.T) {
	svc := mocksports.NewAuthApp(t)
	h := authhandler.NewAuthHandler(svc, mocksjwt.NewSessions(t), mockscookie.NewCookieHandler(t))

	body := buildBody(t, map[string]string{"phone_number": "04141234567"})
	tc := buildhttptest.SetupControllerCase("/auth/riders/identify", http.MethodPost, body)

	err := h.IdentifyRiderPhone(tc.Ctx)

	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestIdentifyRiderPhone_UnexpectedError(t *testing.T) {
	svc := mocksports.NewAuthApp(t)
	h := authhandler.NewAuthHandler(svc, mocksjwt.NewSessions(t), mockscookie.NewCookieHandler(t))

	body := buildBody(t, map[string]string{"phone_number": "+573001112233"})
	tc := buildhttptest.SetupControllerCase("/auth/riders/identify", http.MethodPost, body)
	svc.On("IdentifyRiderPhone", tc.Req.Context(), "+573001112233").Return(models.RiderPhoneNext(""), errcustom.ErrUnexpectedError)

	err := h.IdentifyRiderPhone(tc.Ctx)

	assert.Equal(t, http.StatusInternalServerError, httpCode(err))
}
