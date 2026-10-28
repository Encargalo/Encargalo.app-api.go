package handler_test

import (
	"net/http"
	"strings"
	"testing"

	mocksriders "Encargalo.app-api.go/internal/mocks/riders/domain/ports"
	riderModels "Encargalo.app-api.go/internal/riders/domain/models"
	ridershandler "Encargalo.app-api.go/internal/riders/handler"
	"Encargalo.app-api.go/internal/shared/buildhttptest"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestReportLocation_Success(t *testing.T) {
	svc := mocksriders.NewRidersApp(t)
	h := ridershandler.New(svc)
	riderID := uuid.New()

	body := `{"latitude":10.48,"longitude":-66.90,"mode":"trips"}`
	tc := buildhttptest.SetupControllerCase("/riders/location", http.MethodPost, strings.NewReader(body))
	tc = withRiderSession(tc, riderID)

	svc.On("ReportLocation", tc.Req.Context(), riderID, 10.48, -66.90, riderModels.RiderModeTrips).Return(nil)

	err := h.ReportLocation(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, tc.Res.Code)
}

func TestReportLocation_MissingModeIsRejected(t *testing.T) {
	svc := mocksriders.NewRidersApp(t)
	h := ridershandler.New(svc)
	riderID := uuid.New()

	body := `{"latitude":10.48,"longitude":-66.90}`
	tc := buildhttptest.SetupControllerCase("/riders/location", http.MethodPost, strings.NewReader(body))
	tc = withRiderSession(tc, riderID)

	err := h.ReportLocation(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestReportLocation_InvalidModeIsRejected(t *testing.T) {
	svc := mocksriders.NewRidersApp(t)
	h := ridershandler.New(svc)
	riderID := uuid.New()

	body := `{"latitude":10.48,"longitude":-66.90,"mode":"flying"}`
	tc := buildhttptest.SetupControllerCase("/riders/location", http.MethodPost, strings.NewReader(body))
	tc = withRiderSession(tc, riderID)

	err := h.ReportLocation(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestReportLocation_InvalidCoordinatesMapsTo400(t *testing.T) {
	svc := mocksriders.NewRidersApp(t)
	h := ridershandler.New(svc)
	riderID := uuid.New()

	body := `{"latitude":48.85,"longitude":2.35,"mode":"trips"}`
	tc := buildhttptest.SetupControllerCase("/riders/location", http.MethodPost, strings.NewReader(body))
	tc = withRiderSession(tc, riderID)

	svc.On("ReportLocation", tc.Req.Context(), riderID, 48.85, 2.35, riderModels.RiderModeTrips).Return(errcustom.ErrInvalidCoordinates)

	err := h.ReportLocation(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}
