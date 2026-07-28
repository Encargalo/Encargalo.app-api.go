package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"

	authModels "Encargalo.app-api.go/internal/auth/domain/models"
	mocksshops "Encargalo.app-api.go/internal/mocks/shops/domain/ports"
	"Encargalo.app-api.go/internal/shared/buildhttptest"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/shops/domain/dtos"
	shopshandler "Encargalo.app-api.go/internal/shops/handler"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func newStaffShopHandler(t *testing.T) (shopshandler.StaffShopHandler, *mocksshops.StaffShopApp) {
	t.Helper()
	svc := mocksshops.NewStaffShopApp(t)
	return shopshandler.NewStaffShopHandler(svc), svc
}

func buildBody(t *testing.T, v any) *bytes.Buffer {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return bytes.NewBuffer(b)
}

func withSession(tc buildhttptest.ControllerCase, session authModels.ActiveSession) buildhttptest.ControllerCase {
	ctx := context.WithValue(tc.Req.Context(), authModels.SessionKey, session)
	tc.Req = tc.Req.WithContext(ctx)
	tc.Ctx.SetRequest(tc.Req)
	return tc
}

func TestCreateDraftShop_InvalidPayload(t *testing.T) {
	h, _ := newStaffShopHandler(t)
	tc := buildhttptest.SetupControllerCase("/shops/staff", http.MethodPost, buildBody(t, map[string]string{"name": ""}))

	err := h.CreateDraftShop(tc.Ctx)

	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestCreateDraftShop_Success(t *testing.T) {
	h, svc := newStaffShopHandler(t)
	ownerID := uuid.New()
	body := buildBody(t, dtos.CreateDraftShopRequest{Name: "Pizzería Don Luigi", Type: "Restaurant", OwnerID: ownerID})
	tc := buildhttptest.SetupControllerCase("/shops/staff", http.MethodPost, body)

	svc.On("CreateDraftShop", tc.Req.Context(), dtos.CreateDraftShopRequest{
		Name: "Pizzería Don Luigi", Type: "Restaurant", OwnerID: ownerID,
	}).Return(dtos.StaffShopResponse{Status: "draft", OwnerID: ownerID}, nil)

	err := h.CreateDraftShop(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, tc.Res.Code)
}

func TestActivateShop_IncompleteReturns422WithMissing(t *testing.T) {
	h, svc := newStaffShopHandler(t)
	shopID := uuid.New()
	staffID := uuid.New()
	tc := buildhttptest.SetupControllerCase("/shops/staff/"+shopID.String()+"/activate", http.MethodPost, nil)
	tc = withSession(tc, authModels.ActiveSession{UserID: staffID})
	tc.Ctx.SetParamNames("id")
	tc.Ctx.SetParamValues(shopID.String())

	svc.On("ActivateShop", tc.Req.Context(), shopID, staffID).Return(dtos.StaffShopResponse{}, errcustom.ErrShopOnboardingIncomplete)
	svc.On("GetShop", tc.Req.Context(), shopID).Return(dtos.StaffShopResponse{
		Completion: dtos.ShopCompletionDTO{Missing: []string{"banner", "address"}},
	}, nil)

	err := h.ActivateShop(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, tc.Res.Code)
}

func TestActivateShop_Success(t *testing.T) {
	h, svc := newStaffShopHandler(t)
	shopID := uuid.New()
	staffID := uuid.New()
	tc := buildhttptest.SetupControllerCase("/shops/staff/"+shopID.String()+"/activate", http.MethodPost, nil)
	tc = withSession(tc, authModels.ActiveSession{UserID: staffID})
	tc.Ctx.SetParamNames("id")
	tc.Ctx.SetParamValues(shopID.String())

	svc.On("ActivateShop", tc.Req.Context(), shopID, staffID).Return(dtos.StaffShopResponse{Status: "active"}, nil)

	err := h.ActivateShop(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
}
