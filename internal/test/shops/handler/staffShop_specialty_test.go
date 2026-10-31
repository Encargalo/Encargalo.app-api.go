package handler_test

import (
	"net/http"
	"testing"

	"Encargalo.app-api.go/internal/shared/buildhttptest"
	"Encargalo.app-api.go/internal/shops/domain/dtos"
	shopsmodels "Encargalo.app-api.go/internal/shops/domain/models/shops"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestCreateDraftShop_InvalidSpecialtyRejected(t *testing.T) {
	h, _ := newStaffShopHandler(t)
	ownerID := uuid.New()
	body := buildBody(t, dtos.CreateDraftShopRequest{Name: "Pizzería Don Luigi", Type: "Restaurant", Specialty: "Ferretería", OwnerID: ownerID})
	tc := buildhttptest.SetupControllerCase("/shops/staff", http.MethodPost, body)

	err := h.CreateDraftShop(tc.Ctx)

	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestCreateDraftShop_WithSpecialty_Success(t *testing.T) {
	h, svc := newStaffShopHandler(t)
	ownerID := uuid.New()
	body := buildBody(t, dtos.CreateDraftShopRequest{Name: "Panadería El Trigo", Type: "Supermarket", Specialty: shopsmodels.ShopSpecialtyBakery, OwnerID: ownerID})
	tc := buildhttptest.SetupControllerCase("/shops/staff", http.MethodPost, body)

	svc.On("CreateDraftShop", tc.Req.Context(), dtos.CreateDraftShopRequest{
		Name: "Panadería El Trigo", Type: "Supermarket", Specialty: shopsmodels.ShopSpecialtyBakery, OwnerID: ownerID,
	}).Return(dtos.StaffShopResponse{Status: "draft", OwnerID: ownerID, Specialty: shopsmodels.ShopSpecialtyBakery}, nil)

	err := h.CreateDraftShop(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, tc.Res.Code)
}

func TestUpdateDraftShop_InvalidSpecialtyRejected(t *testing.T) {
	h, _ := newStaffShopHandler(t)
	shopID := uuid.New()
	body := buildBody(t, map[string]string{"specialty": "Ferretería"})
	tc := buildhttptest.SetupControllerCase("/shops/staff/"+shopID.String(), http.MethodPatch, body)
	tc.Ctx.SetParamNames("id")
	tc.Ctx.SetParamValues(shopID.String())

	err := h.UpdateDraftShop(tc.Ctx)

	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestUpdateDraftShop_WithValidSpecialty_Success(t *testing.T) {
	h, svc := newStaffShopHandler(t)
	shopID := uuid.New()
	specialty := shopsmodels.ShopSpecialtyProduce
	// Un cliente real nunca serializa el DTO en sí (Nullable solo define
	// UnmarshalJSON): construye el JSON a mano, igual que lo haría la app.
	body := buildBody(t, map[string]string{"specialty": specialty})
	tc := buildhttptest.SetupControllerCase("/shops/staff/"+shopID.String(), http.MethodPatch, body)
	tc.Ctx.SetParamNames("id")
	tc.Ctx.SetParamValues(shopID.String())

	svc.On("UpdateDraftShop", tc.Req.Context(), shopID, dtos.UpdateDraftShopRequest{
		Specialty: &specialty,
	}).Return(dtos.StaffShopResponse{Specialty: shopsmodels.ShopSpecialtyProduce}, nil)

	err := h.UpdateDraftShop(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
}
