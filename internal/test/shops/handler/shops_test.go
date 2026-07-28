package handler_test

import (
	"net/http"
	"testing"

	mocksshops "Encargalo.app-api.go/internal/mocks/shops/domain/ports"
	"Encargalo.app-api.go/internal/shared/buildhttptest"
	"Encargalo.app-api.go/internal/shops/domain/dtos"
	shopshandler "Encargalo.app-api.go/internal/shops/handler"
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

func newShopsHandler(t *testing.T) (shopshandler.Shops, *mocksshops.ShopsApp) {
	t.Helper()
	app := mocksshops.NewShopsApp(t)
	return shopshandler.NewShopsHandler(app), app
}

func TestGetAllShops_SuccessWithoutType(t *testing.T) {
	h, app := newShopsHandler(t)

	tc := buildhttptest.SetupControllerCase("/shops/all?lat=4.678034&lon=-74.0496399", http.MethodGet, nil)

	app.On("GetAllShops", tc.Req.Context(), mock.MatchedBy(func(q dtos.ListShopsQuery) bool {
		return q.Latitude == 4.678034 && q.Longitude == -74.0496399 && q.Type == ""
	})).Return(dtos.ShopsResponse{}, nil)

	err := h.GetAllShops(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
}

func TestGetAllShops_SuccessWithRestaurant(t *testing.T) {
	h, app := newShopsHandler(t)

	tc := buildhttptest.SetupControllerCase("/shops/all?lat=4.678034&lon=-74.0496399&type=Restaurant", http.MethodGet, nil)

	app.On("GetAllShops", tc.Req.Context(), mock.MatchedBy(func(q dtos.ListShopsQuery) bool {
		return q.Type == "Restaurant"
	})).Return(dtos.ShopsResponse{}, nil)

	err := h.GetAllShops(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
}

func TestGetAllShops_SuccessWithSupermarket(t *testing.T) {
	h, app := newShopsHandler(t)

	tc := buildhttptest.SetupControllerCase("/shops/all?lat=4.678034&lon=-74.0496399&type=Supermarket", http.MethodGet, nil)

	app.On("GetAllShops", tc.Req.Context(), mock.MatchedBy(func(q dtos.ListShopsQuery) bool {
		return q.Type == "Supermarket"
	})).Return(dtos.ShopsResponse{}, nil)

	err := h.GetAllShops(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
}

func TestGetAllShops_InvalidType(t *testing.T) {
	h, _ := newShopsHandler(t)

	tc := buildhttptest.SetupControllerCase("/shops/all?lat=4.678034&lon=-74.0496399&type=Pharmacy", http.MethodGet, nil)

	err := h.GetAllShops(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestSearchShops_SuccessWithoutType(t *testing.T) {
	h, app := newShopsHandler(t)

	tc := buildhttptest.SetupControllerCase("/shops/search?q=arroz&lat=4.678034&lon=-74.0496399", http.MethodGet, nil)

	app.On("SearchShops", tc.Req.Context(), mock.MatchedBy(func(q dtos.SearchShopsQuery) bool {
		return q.Q == "arroz" && q.Type == ""
	})).Return(dtos.SearchShopsResult{}, nil)

	err := h.SearchShops(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
}

func TestSearchShops_SuccessWithSupermarket(t *testing.T) {
	h, app := newShopsHandler(t)

	tc := buildhttptest.SetupControllerCase("/shops/search?q=arroz&lat=4.678034&lon=-74.0496399&type=Supermarket", http.MethodGet, nil)

	app.On("SearchShops", tc.Req.Context(), mock.MatchedBy(func(q dtos.SearchShopsQuery) bool {
		return q.Q == "arroz" && q.Type == "Supermarket"
	})).Return(dtos.SearchShopsResult{}, nil)

	err := h.SearchShops(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
}

func TestSearchShops_InvalidType(t *testing.T) {
	h, _ := newShopsHandler(t)

	tc := buildhttptest.SetupControllerCase("/shops/search?q=arroz&lat=4.678034&lon=-74.0496399&type=Pharmacy", http.MethodGet, nil)

	err := h.SearchShops(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}
