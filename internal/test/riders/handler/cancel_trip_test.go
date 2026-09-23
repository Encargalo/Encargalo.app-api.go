package handler_test

import (
	"context"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	authModels "Encargalo.app-api.go/internal/auth/domain/models"
	mocksriders "Encargalo.app-api.go/internal/mocks/riders/domain/ports"
	ridershandler "Encargalo.app-api.go/internal/riders/handler"
	"Encargalo.app-api.go/internal/shared/config"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	envs := map[string]string{
		"SERVER_PORT":              "3000",
		"POSTGRES_HOST":            "localhost",
		"POSTGRES_PORT":            "5432",
		"POSTGRES_USER":            "test",
		"POSTGRES_PASSWORD":        "test",
		"POSTGRES_DATABASE":        "test",
		"REDIS_URL":                "redis://localhost:6379",
		"REDIS_STREAM_STREAM_NAME": "test-stream",
		"JWT_SECRET":               "test-secret",
		"ENVIRONMENT_CURRENT":      "test",
		"GOOGLE_SERVER_KEY":        "test",
		"GOOGLE_CLIENT_ID":         "test",
		"CLOUDINARY_URL":           "cloudinary://test",
		"CLOUDINARY_FOLDER":        "test",
		"EXTERNAL_API_KEY":         "test",
		"INTERNAL_GATEWAY_SECRET":  "test",
	}
	for k, v := range envs {
		if err := os.Setenv(k, v); err != nil {
			panic(err)
		}
	}
	config.Environments()
	os.Exit(m.Run())
}

func TestHandleWebSocket_CancelTripDispatchesToApp(t *testing.T) {
	svc := mocksriders.NewRidersApp(t)
	h := ridershandler.New(svc)
	riderID := uuid.New()
	tripID := uuid.New()

	e := echo.New()
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx := context.WithValue(c.Request().Context(), authModels.SessionKey, authModels.ActiveSession{UserID: riderID})
			c.SetRequest(c.Request().WithContext(ctx))
			return next(c)
		}
	})
	e.GET("/orders/rider", h.HandleWebSocket)

	svc.On("Register", mock.Anything, riderID, mock.Anything, mock.Anything).Return(nil)
	disconnected := make(chan struct{})
	svc.On("Unregister", riderID, mock.Anything).Run(func(mock.Arguments) {
		close(disconnected)
	}).Return()
	called := make(chan struct{})
	svc.On("CancelAcceptedTrip", mock.Anything, riderID, tripID, mock.Anything).Run(func(mock.Arguments) {
		close(called)
	}).Return(nil)

	server := httptest.NewServer(e)
	t.Cleanup(server.Close)

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/orders/rider"
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	if resp != nil && resp.Body != nil {
		require.NoError(t, resp.Body.Close())
	}

	_, welcome, err := conn.ReadMessage()
	require.NoError(t, err)
	require.JSONEq(t, `{"type":"connected"}`, string(welcome))

	require.NoError(t, conn.WriteJSON(map[string]string{
		"type":    "cancel_trip",
		"trip_id": tripID.String(),
	}))

	select {
	case <-called:
	case <-time.After(2 * time.Second):
		t.Fatal("cancel_trip no llegó a CancelAcceptedTrip")
	}
	require.NoError(t, conn.Close())
	select {
	case <-disconnected:
	case <-time.After(2 * time.Second):
		t.Fatal("el handler no llamó a Unregister al cerrar el socket")
	}
}

func TestHandleWebSocket_CancelTripRejectsInvalidID(t *testing.T) {
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

	svc.On("Register", mock.Anything, riderID, mock.Anything, mock.Anything).Return(nil)
	disconnected := make(chan struct{})
	svc.On("Unregister", riderID, mock.Anything).Run(func(mock.Arguments) {
		close(disconnected)
	}).Return()

	server := httptest.NewServer(e)
	t.Cleanup(server.Close)

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/orders/rider"
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	if resp != nil && resp.Body != nil {
		require.NoError(t, resp.Body.Close())
	}

	_, welcome, err := conn.ReadMessage()
	require.NoError(t, err)
	require.JSONEq(t, `{"type":"connected"}`, string(welcome))

	require.NoError(t, conn.WriteJSON(map[string]string{
		"type":    "cancel_trip",
		"trip_id": "not-a-uuid",
	}))

	require.NoError(t, conn.SetReadDeadline(time.Now().Add(2*time.Second)))
	_, msg, err := conn.ReadMessage()
	require.NoError(t, err)
	require.JSONEq(t, `{"type":"error","message":"trip_id inválido"}`, string(msg))
	svc.AssertNotCalled(t, "CancelAcceptedTrip", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	require.NoError(t, conn.Close())
	select {
	case <-disconnected:
	case <-time.After(2 * time.Second):
		t.Fatal("el handler no llamó a Unregister al cerrar el socket")
	}
}
