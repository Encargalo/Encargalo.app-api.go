package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	authModels "Encargalo.app-api.go/internal/auth/domain/models"
	mocksriders "Encargalo.app-api.go/internal/mocks/riders/domain/ports"
	riderModels "Encargalo.app-api.go/internal/riders/domain/models"
	ridershandler "Encargalo.app-api.go/internal/riders/handler"
	"Encargalo.app-api.go/internal/shared/buildhttptest"
	"Encargalo.app-api.go/internal/shared/errcustom"
	tripsDtos "Encargalo.app-api.go/internal/trips/domain/dtos"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func withRiderSession(tc buildhttptest.ControllerCase, riderID uuid.UUID) buildhttptest.ControllerCase {
	ctx := context.WithValue(tc.Req.Context(), authModels.SessionKey, authModels.ActiveSession{UserID: riderID})
	tc.Req = tc.Req.WithContext(ctx)
	tc.Ctx.SetRequest(tc.Req)
	return tc
}

func TestGetActiveTrips_Success(t *testing.T) {
	svc := mocksriders.NewRidersApp(t)
	h := ridershandler.New(svc)
	riderID := uuid.New()
	tripID := uuid.New()

	tc := buildhttptest.SetupControllerCase("/riders/trips/active", http.MethodGet, nil)
	tc = withRiderSession(tc, riderID)

	svc.On("ActiveTrips", tc.Req.Context(), riderID).Return([]tripsDtos.TripResponse{
		{ID: tripID, Status: "accepted"},
	}, nil)

	err := h.GetActiveTrips(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
	var body struct {
		Trips []struct {
			ID     uuid.UUID `json:"id"`
			Status string    `json:"status"`
		} `json:"trips"`
	}
	require.NoError(t, json.Unmarshal(tc.Res.Body.Bytes(), &body))
	require.Len(t, body.Trips, 1)
	assert.Equal(t, tripID, body.Trips[0].ID)
	assert.Equal(t, "accepted", body.Trips[0].Status)
}

func TestGetActiveTrips_EmptyReturnsEmptyList(t *testing.T) {
	svc := mocksriders.NewRidersApp(t)
	h := ridershandler.New(svc)
	riderID := uuid.New()

	tc := buildhttptest.SetupControllerCase("/riders/trips/active", http.MethodGet, nil)
	tc = withRiderSession(tc, riderID)

	svc.On("ActiveTrips", tc.Req.Context(), riderID).Return([]tripsDtos.TripResponse{}, nil)

	err := h.GetActiveTrips(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
	assert.JSONEq(t, `{"trips":[]}`, tc.Res.Body.String())
}

func TestGetActiveTrips_ServiceError(t *testing.T) {
	svc := mocksriders.NewRidersApp(t)
	h := ridershandler.New(svc)
	riderID := uuid.New()

	tc := buildhttptest.SetupControllerCase("/riders/trips/active", http.MethodGet, nil)
	tc = withRiderSession(tc, riderID)

	svc.On("ActiveTrips", tc.Req.Context(), riderID).Return(nil, errcustom.ErrUnexpectedError)

	err := h.GetActiveTrips(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusInternalServerError, httpCode(err))
}

func TestHandleWebSocket_TripsModeSendsActiveTripsAfterConnected(t *testing.T) {
	svc := mocksriders.NewRidersApp(t)
	h := ridershandler.New(svc)
	riderID := uuid.New()

	e := echo.New()
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx := context.WithValue(c.Request().Context(), authModels.SessionKey, authModels.ActiveSession{UserID: riderID})
			c.SetRequest(c.Request().WithContext(ctx))
			return next(c)
		}
	})
	e.GET("/orders/rider", h.HandleWebSocket)

	svc.On("Register", mock.Anything, riderID, mock.Anything, riderModels.RiderModeTrips).Return(nil)
	replayed := make(chan struct{})
	svc.On("SendActiveTrips", mock.Anything, riderID, mock.Anything).Run(func(mock.Arguments) {
		close(replayed)
	}).Return(nil)
	disconnected := make(chan struct{})
	svc.On("Unregister", riderID, mock.Anything).Run(func(mock.Arguments) {
		close(disconnected)
	}).Return()

	server := httptest.NewServer(e)
	t.Cleanup(server.Close)

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/orders/rider?mode=trips"
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	if resp != nil && resp.Body != nil {
		require.NoError(t, resp.Body.Close())
	}

	_, welcome, err := conn.ReadMessage()
	require.NoError(t, err)
	require.JSONEq(t, `{"type":"connected"}`, string(welcome))

	select {
	case <-replayed:
	case <-time.After(2 * time.Second):
		t.Fatal("el handler no reenvió los viajes activos al conectar en modo trips")
	}
	require.NoError(t, conn.Close())
	select {
	case <-disconnected:
	case <-time.After(2 * time.Second):
		t.Fatal("el handler no llamó a Unregister al cerrar el socket")
	}
}
