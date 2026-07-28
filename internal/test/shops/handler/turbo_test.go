package handler_test

import (
	"net/http"
	"testing"

	"Encargalo.app-api.go/internal/shared/buildhttptest"
	"Encargalo.app-api.go/internal/shops/domain/dtos"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetTurboEntry_Available(t *testing.T) {
	h, app := newShopsHandler(t)

	tc := buildhttptest.SetupControllerCase("/turbo/entry?lat=10.4806&lon=-66.9036", http.MethodGet, nil)

	eta := 8
	shop := dtos.ShopResponse{ID: uuid.New(), Name: "Turbo Fresh", Type: "Turbo", Opened: true, Time: eta}
	app.On("GetTurboEntry", tc.Req.Context(), mock.MatchedBy(func(c dtos.Coords) bool {
		return c.Latitude == 10.4806 && c.Longitude == -66.9036
	})).Return(dtos.TurboEntryResponse{Available: true, Shop: &shop, EtaMinutes: &eta, CoverageKm: 2}, nil)

	err := h.GetTurboEntry(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
}

func TestGetTurboEntry_NotAvailable(t *testing.T) {
	h, app := newShopsHandler(t)

	tc := buildhttptest.SetupControllerCase("/turbo/entry?lat=10.4806&lon=-66.9036", http.MethodGet, nil)

	app.On("GetTurboEntry", tc.Req.Context(), mock.Anything).Return(dtos.TurboEntryResponse{Available: false}, nil)

	err := h.GetTurboEntry(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
}

func TestGetTurboEntry_InvalidCoords(t *testing.T) {
	h, _ := newShopsHandler(t)

	tc := buildhttptest.SetupControllerCase("/turbo/entry?lat=999&lon=-66.9036", http.MethodGet, nil)

	err := h.GetTurboEntry(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestGetTurboEntry_MissingCoords(t *testing.T) {
	h, _ := newShopsHandler(t)

	tc := buildhttptest.SetupControllerCase("/turbo/entry", http.MethodGet, nil)

	err := h.GetTurboEntry(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}
