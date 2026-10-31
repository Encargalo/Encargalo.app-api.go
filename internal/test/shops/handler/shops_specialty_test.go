package handler_test

import (
	"net/http"
	"testing"

	"Encargalo.app-api.go/internal/shared/buildhttptest"
	"Encargalo.app-api.go/internal/shops/domain/dtos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetAllShops_SuccessWithSpecialty(t *testing.T) {
	h, app := newShopsHandler(t)

	tc := buildhttptest.SetupControllerCase("/shops/all?lat=4.678034&lon=-74.0496399&type=Supermarket&specialty=General", http.MethodGet, nil)

	app.On("GetAllShops", tc.Req.Context(), mock.MatchedBy(func(q dtos.ListShopsQuery) bool {
		return q.Type == "Supermarket" && q.Specialty == "General"
	})).Return(dtos.ShopsResponse{}, nil)

	err := h.GetAllShops(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
}

func TestGetAllShops_InvalidSpecialty(t *testing.T) {
	h, _ := newShopsHandler(t)

	tc := buildhttptest.SetupControllerCase("/shops/all?lat=4.678034&lon=-74.0496399&specialty=Bogus", http.MethodGet, nil)

	err := h.GetAllShops(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestSearchShops_SuccessWithSpecialty(t *testing.T) {
	h, app := newShopsHandler(t)

	tc := buildhttptest.SetupControllerCase("/shops/search?q=arroz&lat=4.678034&lon=-74.0496399&type=Supermarket&specialty=General", http.MethodGet, nil)

	app.On("SearchShops", tc.Req.Context(), mock.MatchedBy(func(q dtos.SearchShopsQuery) bool {
		return q.Type == "Supermarket" && q.Specialty == "General"
	})).Return(dtos.SearchShopsResult{}, nil)

	err := h.SearchShops(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
}

func TestSearchShops_InvalidSpecialty(t *testing.T) {
	h, _ := newShopsHandler(t)

	tc := buildhttptest.SetupControllerCase("/shops/search?q=arroz&lat=4.678034&lon=-74.0496399&specialty=Bogus", http.MethodGet, nil)

	err := h.SearchShops(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}
