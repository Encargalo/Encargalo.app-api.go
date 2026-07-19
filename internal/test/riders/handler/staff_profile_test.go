package handler_test

import (
	"net/http"
	"testing"

	mocksriders "Encargalo.app-api.go/internal/mocks/riders/domain/ports"
	"Encargalo.app-api.go/internal/riders/domain/dtos"
	ridershandler "Encargalo.app-api.go/internal/riders/handler"
	"Encargalo.app-api.go/internal/shared/buildhttptest"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func httpCode(err error) int {
	if he, ok := err.(*echo.HTTPError); ok {
		return he.Code
	}
	return 0
}

func TestDeletePhoto_Success(t *testing.T) {
	svc := mocksriders.NewRiderProfileApp(t)
	h := ridershandler.NewStaffRiderHandler(svc)

	userID := uuid.New()
	tc := buildhttptest.SetupControllerCase(
		"/riders/staff/profiles/"+userID.String()+"/photos/id_photo",
		http.MethodDelete,
		nil,
	)
	tc.Ctx.SetParamNames("userId", "field")
	tc.Ctx.SetParamValues(userID.String(), "id_photo")

	svc.On("DeletePhoto", mock.Anything, userID, "id_photo").Return(dtos.RiderProfileResponse{
		UserID: userID.String(),
		Status: "draft",
	}, nil)

	err := h.DeletePhoto(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
}

func TestDeletePhoto_InvalidUserID(t *testing.T) {
	svc := mocksriders.NewRiderProfileApp(t)
	h := ridershandler.NewStaffRiderHandler(svc)

	tc := buildhttptest.SetupControllerCase(
		"/riders/staff/profiles/not-a-uuid/photos/id_photo",
		http.MethodDelete,
		nil,
	)
	tc.Ctx.SetParamNames("userId", "field")
	tc.Ctx.SetParamValues("not-a-uuid", "id_photo")

	err := h.DeletePhoto(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestDeletePhoto_InvalidField(t *testing.T) {
	svc := mocksriders.NewRiderProfileApp(t)
	h := ridershandler.NewStaffRiderHandler(svc)

	userID := uuid.New()
	tc := buildhttptest.SetupControllerCase(
		"/riders/staff/profiles/"+userID.String()+"/photos/not_a_field",
		http.MethodDelete,
		nil,
	)
	tc.Ctx.SetParamNames("userId", "field")
	tc.Ctx.SetParamValues(userID.String(), "not_a_field")

	err := h.DeletePhoto(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestDeletePhoto_NotFound(t *testing.T) {
	svc := mocksriders.NewRiderProfileApp(t)
	h := ridershandler.NewStaffRiderHandler(svc)

	userID := uuid.New()
	tc := buildhttptest.SetupControllerCase(
		"/riders/staff/profiles/"+userID.String()+"/photos/profile_photo",
		http.MethodDelete,
		nil,
	)
	tc.Ctx.SetParamNames("userId", "field")
	tc.Ctx.SetParamValues(userID.String(), "profile_photo")

	svc.On("DeletePhoto", mock.Anything, userID, "profile_photo").Return(
		dtos.RiderProfileResponse{},
		errcustom.ErrNotFound,
	)

	err := h.DeletePhoto(tc.Ctx)

	assert.Error(t, err)
}
