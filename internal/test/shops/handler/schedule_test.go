package handler_test

import (
	"net/http"
	"testing"

	authModels "Encargalo.app-api.go/internal/auth/domain/models"
	"Encargalo.app-api.go/internal/shared/buildhttptest"
	"Encargalo.app-api.go/internal/shops/domain/dtos"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetMySchedule_NoShopAssociated(t *testing.T) {
	h, _ := newShopsHandler(t)
	tc := buildhttptest.SetupControllerCase("/business/shops/me/schedule", http.MethodGet, nil)
	tc = withSession(tc, authModels.ActiveSession{UserID: uuid.New()})

	err := h.GetMySchedule(tc.Ctx)

	assert.Equal(t, http.StatusUnprocessableEntity, httpCode(err))
}

func TestGetMySchedule_Success(t *testing.T) {
	h, app := newShopsHandler(t)
	shopID := uuid.New()
	tc := buildhttptest.SetupControllerCase("/business/shops/me/schedule", http.MethodGet, nil)
	tc = withSession(tc, authModels.ActiveSession{UserID: uuid.New(), ShopID: &shopID})

	app.On("GetMySchedule", tc.Req.Context(), shopID).Return([]dtos.ScheduleResponse{
		{DayOfWeek: 1, Open: "08:00:00", Close: "20:00:00", Enabled: true},
	}, nil)

	err := h.GetMySchedule(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
}

func TestSetMySchedule_InvalidPayload(t *testing.T) {
	h, _ := newShopsHandler(t)
	shopID := uuid.New()
	tc := buildhttptest.SetupControllerCase("/business/shops/me/schedule", http.MethodPut,
		buildBody(t, dtos.ShopScheduleRequest{Schedules: []dtos.ScheduleInput{{DayOfWeek: 1, Open: "08:00:00", Close: "08:00:00"}}}))
	tc = withSession(tc, authModels.ActiveSession{UserID: uuid.New(), ShopID: &shopID})

	err := h.SetMySchedule(tc.Ctx)

	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestSetMySchedule_Success(t *testing.T) {
	h, app := newShopsHandler(t)
	shopID := uuid.New()
	req := dtos.ShopScheduleRequest{Schedules: []dtos.ScheduleInput{{DayOfWeek: 1, Open: "08:00:00", Close: "20:00:00"}}}
	tc := buildhttptest.SetupControllerCase("/business/shops/me/schedule", http.MethodPut, buildBody(t, req))
	tc = withSession(tc, authModels.ActiveSession{UserID: uuid.New(), ShopID: &shopID})

	app.On("SetMySchedule", tc.Req.Context(), shopID, req.Schedules).Return(nil)

	err := h.SetMySchedule(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, tc.Res.Code)
}

func TestStaffGetShopSchedule_Success(t *testing.T) {
	h, svc := newStaffShopHandler(t)
	shopID := uuid.New()
	tc := buildhttptest.SetupControllerCase("/shops/staff/"+shopID.String()+"/schedule", http.MethodGet, nil)
	tc.Ctx.SetParamNames("id")
	tc.Ctx.SetParamValues(shopID.String())

	svc.On("GetShopSchedule", mock.Anything, shopID).Return([]dtos.ScheduleResponse{}, nil)

	err := h.GetShopSchedule(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
}

func TestStaffSetShopSchedule_InvalidID(t *testing.T) {
	h, _ := newStaffShopHandler(t)
	tc := buildhttptest.SetupControllerCase("/shops/staff/not-a-uuid/schedule", http.MethodPut,
		buildBody(t, dtos.ShopScheduleRequest{}))
	tc.Ctx.SetParamNames("id")
	tc.Ctx.SetParamValues("not-a-uuid")

	err := h.SetShopSchedule(tc.Ctx)

	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestStaffSetShopSchedule_Success(t *testing.T) {
	h, svc := newStaffShopHandler(t)
	shopID := uuid.New()
	req := dtos.ShopScheduleRequest{Schedules: []dtos.ScheduleInput{{DayOfWeek: 3, Open: "09:00:00", Close: "18:00:00"}}}
	tc := buildhttptest.SetupControllerCase("/shops/staff/"+shopID.String()+"/schedule", http.MethodPut, buildBody(t, req))
	tc.Ctx.SetParamNames("id")
	tc.Ctx.SetParamValues(shopID.String())

	svc.On("SetShopSchedule", mock.Anything, shopID, req.Schedules).Return(nil)

	err := h.SetShopSchedule(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, tc.Res.Code)
}
