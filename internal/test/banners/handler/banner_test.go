package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"Encargalo.app-api.go/internal/banners/domain/dtos"
	bannershandler "Encargalo.app-api.go/internal/banners/handler"
	mocksbanners "Encargalo.app-api.go/internal/mocks/banners/domain/ports"
	"Encargalo.app-api.go/internal/shared/buildhttptest"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func httpCode(err error) int {
	if he, ok := err.(*echo.HTTPError); ok {
		return he.Code
	}
	return 0
}

func buildBody(t *testing.T, v any) *bytes.Buffer {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return bytes.NewBuffer(b)
}

func newStaffBannersHandler(t *testing.T) (bannershandler.StaffBannersHandler, *mocksbanners.BannersApp) {
	t.Helper()
	svc := mocksbanners.NewBannersApp(t)
	return bannershandler.NewStaffBannersHandler(svc), svc
}

func TestListActiveBanners_Success(t *testing.T) {
	svc := mocksbanners.NewBannersApp(t)
	h := bannershandler.NewBannersHandler(svc)
	tc := buildhttptest.SetupControllerCase("/banners", http.MethodGet, nil)

	svc.On("ListActiveBanners", tc.Req.Context()).Return(dtos.BannersResponse{{ID: uuid.New(), ImageURL: "img"}}, nil)

	err := h.ListActiveBanners(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
}

func TestCreateBanner_InvalidPayload(t *testing.T) {
	h, _ := newStaffBannersHandler(t)
	tc := buildhttptest.SetupControllerCase("/banners/staff", http.MethodPost, buildBody(t, map[string]string{"image": ""}))

	err := h.CreateBanner(tc.Ctx)

	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestCreateBanner_Success(t *testing.T) {
	h, svc := newStaffBannersHandler(t)
	body := buildBody(t, dtos.CreateBannerRequest{Image: "data:image/png;base64,AAA"})
	tc := buildhttptest.SetupControllerCase("/banners/staff", http.MethodPost, body)

	svc.On("CreateBanner", tc.Req.Context(), dtos.CreateBannerRequest{Image: "data:image/png;base64,AAA"}).
		Return(dtos.BannerResponse{ImageURL: "img"}, nil)

	err := h.CreateBanner(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, tc.Res.Code)
}

func TestGetBanner_NotFound(t *testing.T) {
	h, svc := newStaffBannersHandler(t)
	bannerID := uuid.New()
	tc := buildhttptest.SetupControllerCase("/banners/staff/"+bannerID.String(), http.MethodGet, nil)
	tc.Ctx.SetParamNames("id")
	tc.Ctx.SetParamValues(bannerID.String())

	svc.On("GetBanner", tc.Req.Context(), bannerID).Return(dtos.BannerResponse{}, errcustom.ErrNotFound)

	err := h.GetBanner(tc.Ctx)

	assert.Equal(t, http.StatusNotFound, httpCode(err))
}

func TestUpdateBanner_InvalidID(t *testing.T) {
	h, _ := newStaffBannersHandler(t)
	tc := buildhttptest.SetupControllerCase("/banners/staff/not-a-uuid", http.MethodPatch, buildBody(t, map[string]string{}))
	tc.Ctx.SetParamNames("id")
	tc.Ctx.SetParamValues("not-a-uuid")

	err := h.UpdateBanner(tc.Ctx)

	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestDeleteBanner_Success(t *testing.T) {
	h, svc := newStaffBannersHandler(t)
	bannerID := uuid.New()
	tc := buildhttptest.SetupControllerCase("/banners/staff/"+bannerID.String(), http.MethodDelete, nil)
	tc.Ctx.SetParamNames("id")
	tc.Ctx.SetParamValues(bannerID.String())

	svc.On("DeleteBanner", tc.Req.Context(), bannerID).Return(nil)

	err := h.DeleteBanner(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, tc.Res.Code)
}
