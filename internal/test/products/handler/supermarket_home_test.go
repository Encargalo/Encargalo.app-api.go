package handler_test

import (
	"net/http"
	"testing"

	"Encargalo.app-api.go/internal/products/domain/dtos"
	"Encargalo.app-api.go/internal/shared/buildhttptest"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetSupermarketHome_Success(t *testing.T) {
	h, svc := newHandler(t)

	tc := buildhttptest.SetupControllerCase("/products/home/supermarket?lat=4.678034&lon=-74.0496399", http.MethodGet, nil)

	svc.On("GetSupermarketHome", tc.Req.Context(), mock.MatchedBy(func(q dtos.SupermarketHomeCoords) bool {
		return q.Latitude == 4.678034 && q.Longitude == -74.0496399
	})).Return(dtos.SupermarketHomeResponse{}, nil)

	err := h.GetSupermarketHome(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
}

func TestGetSupermarketHome_MissingCoords(t *testing.T) {
	h, _ := newHandler(t)

	tc := buildhttptest.SetupControllerCase("/products/home/supermarket", http.MethodGet, nil)

	err := h.GetSupermarketHome(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestGetSupermarketHome_InvalidCoords(t *testing.T) {
	h, _ := newHandler(t)

	tc := buildhttptest.SetupControllerCase("/products/home/supermarket?lat=999&lon=-74.0496399", http.MethodGet, nil)

	err := h.GetSupermarketHome(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestGetSupermarketHome_ServiceError(t *testing.T) {
	h, svc := newHandler(t)

	tc := buildhttptest.SetupControllerCase("/products/home/supermarket?lat=4.678034&lon=-74.0496399", http.MethodGet, nil)

	svc.On("GetSupermarketHome", tc.Req.Context(), mock.Anything).Return(dtos.SupermarketHomeResponse(nil), errcustom.ErrUnexpectedError)

	err := h.GetSupermarketHome(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, errcustom.ErrUnexpectedError, err)
}
