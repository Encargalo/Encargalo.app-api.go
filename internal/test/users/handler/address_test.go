package handler_test

import (
	"context"
	"net/http"
	"testing"

	authmodels "Encargalo.app-api.go/internal/auth/domain/models"
	mocksusersports "Encargalo.app-api.go/internal/mocks/users/domain/ports"
	"Encargalo.app-api.go/internal/shared/buildhttptest"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/users/domain/dto"
	usermodels "Encargalo.app-api.go/internal/users/domain/models"
	usershandler "Encargalo.app-api.go/internal/users/handler"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func newAddressHandler(t *testing.T) (usershandler.UsersAddressHandler, *mocksusersports.UsersAddressApp) {
	t.Helper()
	svc := mocksusersports.NewUsersAddressApp(t)
	h := usershandler.NewUsersAddressHandler(svc)
	return h, svc
}

func addressSessionCtx(t *testing.T, tc buildhttptest.ControllerCase, userID uuid.UUID) (context.Context, buildhttptest.ControllerCase) {
	t.Helper()
	session := authmodels.ActiveSession{ID: uuid.New(), UserID: userID}
	ctx := context.WithValue(tc.Req.Context(), authmodels.SessionKey, session)
	tc.Req = tc.Req.WithContext(ctx)
	tc.Ctx.SetRequest(tc.Req)
	return ctx, tc
}

// --- RegisterAddress ---

func TestRegisterAddress_Success(t *testing.T) {
	h, svc := newAddressHandler(t)

	userID := uuid.New()
	body := buildBody(t, map[string]any{
		"type": "Home", "label": "Casa", "address": "Calle 123",
		"lat": 4.60971, "long": -74.08175,
		"extra": map[string]string{"details": "Casa de dos pisos"},
	})
	tc := buildhttptest.SetupControllerCase("/users/address", http.MethodPost, body)
	ctx, tc := addressSessionCtx(t, tc, userID)

	svc.On("RegisterAddress", ctx, userID, dto.Address{
		Type: "Home", Label: "Casa", Address: "Calle 123",
		Latitude: 4.60971, Longitude: -74.08175,
		Extra: &dto.Home{Details: "Casa de dos pisos"},
	}).Return(nil)

	err := h.RegisterAddress(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, tc.Res.Code)
}

func TestRegisterAddress_InvalidBody(t *testing.T) {
	h, _ := newAddressHandler(t)

	body := buildBody(t, map[string]any{
		"type": "Unknown", "label": "X", "address": "Y",
		"lat": 4.60971, "long": -74.08175, "extra": nil,
	})
	tc := buildhttptest.SetupControllerCase("/users/address", http.MethodPost, body)

	err := h.RegisterAddress(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestRegisterAddress_ValidationFail(t *testing.T) {
	h, _ := newAddressHandler(t)

	// lat/long fuera de rango
	body := buildBody(t, map[string]any{
		"type": "Home", "label": "", "address": "",
		"lat": 999.0, "long": 999.0,
		"extra": map[string]string{},
	})
	tc := buildhttptest.SetupControllerCase("/users/address", http.MethodPost, body)

	err := h.RegisterAddress(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestRegisterAddress_NoSession(t *testing.T) {
	h, _ := newAddressHandler(t)

	body := buildBody(t, map[string]any{
		"type": "Home", "label": "Casa", "address": "Calle 123",
		"lat": 4.60971, "long": -74.08175,
		"extra": map[string]string{"details": "Casa de dos pisos"},
	})
	tc := buildhttptest.SetupControllerCase("/users/address", http.MethodPost, body)

	err := h.RegisterAddress(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, errcustom.ErrUnexpectedError, err)
}

func TestRegisterAddress_UnexpectedError(t *testing.T) {
	h, svc := newAddressHandler(t)

	userID := uuid.New()
	body := buildBody(t, map[string]any{
		"type": "Home", "label": "Casa", "address": "Calle 123",
		"lat": 4.60971, "long": -74.08175,
		"extra": map[string]string{"details": "Casa de dos pisos"},
	})
	tc := buildhttptest.SetupControllerCase("/users/address", http.MethodPost, body)
	ctx, tc := addressSessionCtx(t, tc, userID)

	svc.On("RegisterAddress", ctx, userID, dto.Address{
		Type: "Home", Label: "Casa", Address: "Calle 123",
		Latitude: 4.60971, Longitude: -74.08175,
		Extra: &dto.Home{Details: "Casa de dos pisos"},
	}).Return(errcustom.ErrUnexpectedError)

	err := h.RegisterAddress(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusInternalServerError, httpCode(err))
}

// --- SearchAllAddress ---

func TestSearchAllAddress_Success(t *testing.T) {
	h, svc := newAddressHandler(t)

	userID := uuid.New()
	tc := buildhttptest.SetupControllerCase("/users/address", http.MethodGet, nil)
	ctx, tc := addressSessionCtx(t, tc, userID)

	addresses := usermodels.Addresses{}
	svc.On("SearchAllAddress", ctx, userID).Return(addresses, nil)

	err := h.SearchAllAdrress(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
}

func TestSearchAllAddress_NoSession(t *testing.T) {
	h, _ := newAddressHandler(t)

	tc := buildhttptest.SetupControllerCase("/users/address", http.MethodGet, nil)

	err := h.SearchAllAdrress(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, errcustom.ErrUnexpectedError, err)
}

func TestSearchAllAddress_UnexpectedError(t *testing.T) {
	h, svc := newAddressHandler(t)

	userID := uuid.New()
	tc := buildhttptest.SetupControllerCase("/users/address", http.MethodGet, nil)
	ctx, tc := addressSessionCtx(t, tc, userID)

	svc.On("SearchAllAddress", ctx, userID).Return(usermodels.Addresses(nil), errcustom.ErrUnexpectedError)

	err := h.SearchAllAdrress(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusInternalServerError, httpCode(err))
}

// --- DeleteAddress ---

func TestDeleteAddress_Success(t *testing.T) {
	h, svc := newAddressHandler(t)

	userID := uuid.New()
	addressID := uuid.New().String()
	tc := buildhttptest.SetupControllerCase("/users/address?address_id="+addressID, http.MethodDelete, nil)
	tc.Ctx.QueryParams().Set("address_id", addressID)
	ctx, tc := addressSessionCtx(t, tc, userID)

	svc.On("DeleteAddress", ctx, userID, addressID).Return(nil)

	err := h.DeleteAddress(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
}

func TestDeleteAddress_MissingAddressID(t *testing.T) {
	h, _ := newAddressHandler(t)

	tc := buildhttptest.SetupControllerCase("/users/address", http.MethodDelete, nil)

	err := h.DeleteAddress(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestDeleteAddress_NoSession(t *testing.T) {
	h, _ := newAddressHandler(t)

	addressID := uuid.New().String()
	tc := buildhttptest.SetupControllerCase("/users/address?address_id="+addressID, http.MethodDelete, nil)
	tc.Ctx.QueryParams().Set("address_id", addressID)

	err := h.DeleteAddress(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, errcustom.ErrUnexpectedError, err)
}

func TestDeleteAddress_UnexpectedError(t *testing.T) {
	h, svc := newAddressHandler(t)

	userID := uuid.New()
	addressID := uuid.New().String()
	tc := buildhttptest.SetupControllerCase("/users/address?address_id="+addressID, http.MethodDelete, nil)
	tc.Ctx.QueryParams().Set("address_id", addressID)
	ctx, tc := addressSessionCtx(t, tc, userID)

	svc.On("DeleteAddress", ctx, userID, addressID).Return(errcustom.ErrUnexpectedError)

	err := h.DeleteAddress(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusInternalServerError, httpCode(err))
}
