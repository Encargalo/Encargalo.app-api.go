package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"

	authmodels "Encargalo.app-api.go/internal/auth/domain/models"
	mockstracking "Encargalo.app-api.go/internal/mocks/tracking/domain/ports"
	"Encargalo.app-api.go/internal/shared/buildhttptest"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/tracking/domain/dtos"
	trackinghandler "Encargalo.app-api.go/internal/tracking/handler"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func buildBody(t *testing.T, v any) *bytes.Buffer {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return bytes.NewBuffer(b)
}

func httpCode(err error) int {
	if he, ok := err.(*echo.HTTPError); ok {
		return he.Code
	}
	return 0
}

func newHandler(t *testing.T) (trackinghandler.TrackingHandler, *mockstracking.TrackingApp) {
	t.Helper()
	svc := mockstracking.NewTrackingApp(t)
	return trackinghandler.NewTrackingHandler(svc), svc
}

func withSession(tc buildhttptest.ControllerCase, session authmodels.ActiveSession) buildhttptest.ControllerCase {
	ctx := context.WithValue(tc.Req.Context(), authmodels.SessionKey, session)
	tc.Req = tc.Req.WithContext(ctx)
	tc.Ctx.SetRequest(tc.Req)
	return tc
}

func TestTrack_Success(t *testing.T) {
	h, svc := newHandler(t)

	session := authmodels.ActiveSession{ID: uuid.New(), UserID: uuid.New()}
	event := dtos.TrackEvent{
		ID:     uuid.New(),
		Type:   "pixel",
		Params: map[string]any{"name": "app_open"},
	}
	tc := buildhttptest.SetupControllerCase("/tracking", http.MethodPost, buildBody(t, event))
	tc = withSession(tc, session)

	svc.On("Track", tc.Req.Context(), session.UserID, session.ID, mock.AnythingOfType("TrackEvent")).Return(nil)

	err := h.Track(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, tc.Res.Code)
}

func TestTrack_NoSession(t *testing.T) {
	h, _ := newHandler(t)

	event := dtos.TrackEvent{ID: uuid.New(), Type: "pixel"}
	tc := buildhttptest.SetupControllerCase("/tracking", http.MethodPost, buildBody(t, event))

	err := h.Track(tc.Ctx)

	assert.ErrorIs(t, err, errcustom.ErrUnautorized)
}

func TestTrack_InvalidJSON(t *testing.T) {
	h, _ := newHandler(t)

	session := authmodels.ActiveSession{ID: uuid.New(), UserID: uuid.New()}
	tc := buildhttptest.SetupControllerCase("/tracking", http.MethodPost, bytes.NewBufferString("{"))
	tc = withSession(tc, session)

	err := h.Track(tc.Ctx)

	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestTrack_MissingType(t *testing.T) {
	h, _ := newHandler(t)

	session := authmodels.ActiveSession{ID: uuid.New(), UserID: uuid.New()}
	body := map[string]any{"id": uuid.New().String()}
	tc := buildhttptest.SetupControllerCase("/tracking", http.MethodPost, buildBody(t, body))
	tc = withSession(tc, session)

	err := h.Track(tc.Ctx)

	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestTrack_AppError(t *testing.T) {
	h, svc := newHandler(t)

	session := authmodels.ActiveSession{ID: uuid.New(), UserID: uuid.New()}
	event := dtos.TrackEvent{ID: uuid.New(), Type: "pixel"}
	tc := buildhttptest.SetupControllerCase("/tracking", http.MethodPost, buildBody(t, event))
	tc = withSession(tc, session)

	svc.On("Track", tc.Req.Context(), session.UserID, session.ID, mock.AnythingOfType("TrackEvent")).
		Return(errcustom.ErrUnexpectedError)

	err := h.Track(tc.Ctx)

	assert.ErrorIs(t, err, errcustom.ErrUnexpectedError)
}

func TestTrack_InvalidTypeFromApp(t *testing.T) {
	h, svc := newHandler(t)

	session := authmodels.ActiveSession{ID: uuid.New(), UserID: uuid.New()}
	event := dtos.TrackEvent{ID: uuid.New(), Type: "PIXEL"}
	tc := buildhttptest.SetupControllerCase("/tracking", http.MethodPost, buildBody(t, event))
	tc = withSession(tc, session)

	svc.On("Track", tc.Req.Context(), session.UserID, session.ID, mock.AnythingOfType("TrackEvent")).
		Return(errcustom.ErrInvalidTrackingType)

	err := h.Track(tc.Ctx)

	assert.ErrorIs(t, err, errcustom.ErrInvalidTrackingType)
}
