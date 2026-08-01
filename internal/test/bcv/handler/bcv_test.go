package handler_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"Encargalo.app-api.go/internal/bcv/domain/models"
	bcvhandler "Encargalo.app-api.go/internal/bcv/handler"
	mocksbcv "Encargalo.app-api.go/internal/mocks/bcv/domain/ports"
	"Encargalo.app-api.go/internal/shared/buildhttptest"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/stretchr/testify/assert"
)

func newHandler(t *testing.T) (bcvhandler.BcvHandler, *mocksbcv.BcvApp) {
	t.Helper()
	svc := mocksbcv.NewBcvApp(t)
	return bcvhandler.NewBcvHandler(svc), svc
}

func TestGetRate_Success(t *testing.T) {
	h, svc := newHandler(t)

	rate := &models.ExchangeRate{
		Rate:      52.34,
		Date:      time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
		FetchedAt: time.Date(2026, 8, 1, 9, 10, 0, 0, time.UTC),
	}
	tc := buildhttptest.SetupControllerCase("/bcv/rate", http.MethodGet, nil)

	svc.On("GetRate", tc.Req.Context()).Return(rate, nil)

	err := h.GetRate(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)

	var got models.ExchangeRate
	assert.NoError(t, json.Unmarshal(tc.Res.Body.Bytes(), &got))
	assert.Equal(t, rate.Rate, got.Rate)
}

func TestGetRate_AppError(t *testing.T) {
	h, svc := newHandler(t)

	tc := buildhttptest.SetupControllerCase("/bcv/rate", http.MethodGet, nil)

	svc.On("GetRate", tc.Req.Context()).Return(nil, errors.New("redis down"))

	err := h.GetRate(tc.Ctx)

	assert.ErrorIs(t, err, errcustom.ErrUnexpectedError)
}
