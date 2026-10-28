package handler_test

import (
	"net/http"
	"testing"

	mocksriders "Encargalo.app-api.go/internal/mocks/riders/domain/ports"
	"Encargalo.app-api.go/internal/riders/domain/dtos"
	"Encargalo.app-api.go/internal/riders/domain/models"
	ridershandler "Encargalo.app-api.go/internal/riders/handler"
	"Encargalo.app-api.go/internal/shared/buildhttptest"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestUpsertDeviceToken_Success(t *testing.T) {
	svc := mocksriders.NewRiderProfileApp(t)
	h := ridershandler.NewRiderProfileHandler(svc)
	riderID := uuid.New()

	tc := buildhttptest.SetupControllerCase("/riders/device-token", http.MethodPut, buildBody(t, map[string]any{
		"fcm_token": "abc123",
		"platform":  models.DevicePlatformAndroid,
	}))
	tc = withRiderSession(tc, riderID)

	svc.On("UpsertDeviceToken", tc.Req.Context(), riderID, dtos.UpsertDeviceTokenRequest{
		FCMToken: "abc123",
		Platform: models.DevicePlatformAndroid,
	}).Return(nil)

	err := h.UpsertDeviceToken(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, tc.Res.Code)
}

func TestUpsertDeviceToken_MissingPlatformIsRejected(t *testing.T) {
	svc := mocksriders.NewRiderProfileApp(t)
	h := ridershandler.NewRiderProfileHandler(svc)
	riderID := uuid.New()

	tc := buildhttptest.SetupControllerCase("/riders/device-token", http.MethodPut, buildBody(t, map[string]any{
		"fcm_token": "abc123",
	}))
	tc = withRiderSession(tc, riderID)

	err := h.UpsertDeviceToken(tc.Ctx)

	assert.Error(t, err)
}

func TestUpsertDeviceToken_ServiceErrorMapsToHTTPError(t *testing.T) {
	svc := mocksriders.NewRiderProfileApp(t)
	h := ridershandler.NewRiderProfileHandler(svc)
	riderID := uuid.New()

	tc := buildhttptest.SetupControllerCase("/riders/device-token", http.MethodPut, buildBody(t, map[string]any{
		"fcm_token": "abc123",
		"platform":  models.DevicePlatformIOS,
	}))
	tc = withRiderSession(tc, riderID)

	svc.On("UpsertDeviceToken", tc.Req.Context(), riderID, dtos.UpsertDeviceTokenRequest{
		FCMToken: "abc123",
		Platform: models.DevicePlatformIOS,
	}).Return(errcustom.ErrUnexpectedError)

	err := h.UpsertDeviceToken(tc.Ctx)

	assert.Error(t, err)
}
