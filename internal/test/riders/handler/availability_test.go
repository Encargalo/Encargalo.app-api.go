package handler_test

import (
	"net/http"
	"strings"
	"testing"

	mocksriders "Encargalo.app-api.go/internal/mocks/riders/domain/ports"
	ridershandler "Encargalo.app-api.go/internal/riders/handler"
	"Encargalo.app-api.go/internal/shared/buildhttptest"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestSetAvailability_Success(t *testing.T) {
	svc := mocksriders.NewRidersApp(t)
	h := ridershandler.New(svc)
	riderID := uuid.New()

	body := `{"accepting_new_work":false}`
	tc := buildhttptest.SetupControllerCase("/riders/availability", http.MethodPut, strings.NewReader(body))
	tc = withRiderSession(tc, riderID)

	svc.On("SetAvailability", tc.Req.Context(), riderID, false).Return(nil)

	err := h.SetAvailability(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, tc.Res.Code)
}

func TestSetAvailability_InvalidPayloadIsRejected(t *testing.T) {
	svc := mocksriders.NewRidersApp(t)
	h := ridershandler.New(svc)
	riderID := uuid.New()

	body := `{"accepting_new_work":"not-a-bool"}`
	tc := buildhttptest.SetupControllerCase("/riders/availability", http.MethodPut, strings.NewReader(body))
	tc = withRiderSession(tc, riderID)

	err := h.SetAvailability(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestSetAvailability_AppErrorMapsToHTTP(t *testing.T) {
	svc := mocksriders.NewRidersApp(t)
	h := ridershandler.New(svc)
	riderID := uuid.New()

	body := `{"accepting_new_work":true}`
	tc := buildhttptest.SetupControllerCase("/riders/availability", http.MethodPut, strings.NewReader(body))
	tc = withRiderSession(tc, riderID)

	svc.On("SetAvailability", tc.Req.Context(), riderID, true).Return(errcustom.ErrUnexpectedError)

	err := h.SetAvailability(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusInternalServerError, httpCode(err))
}
