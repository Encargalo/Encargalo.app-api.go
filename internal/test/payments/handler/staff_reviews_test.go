package handler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"Encargalo.app-api.go/internal/auth/domain/models"
	mockpayments "Encargalo.app-api.go/internal/mocks/payments/domain/ports"
	paymentsmodels "Encargalo.app-api.go/internal/payments/domain/models"
	"Encargalo.app-api.go/internal/payments/handler"
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

func TestStaffReviewsHandler_ApproveReturnsReference(t *testing.T) {
	hub := mockpayments.NewStaffReviewHub(t)
	reviews := mockpayments.NewPaymentReviewsApp(t)
	h := handler.NewStaffReviewsHandler(hub, reviews)

	staffID := uuid.New()
	reviewID := uuid.New()
	referenceID := uuid.New()
	hub.On("Register", mock.Anything, staffID, mock.Anything).Return(nil)
	unregistered := make(chan struct{})
	hub.On("Unregister", staffID, mock.Anything).Run(func(mock.Arguments) {
		close(unregistered)
	}).Return()
	reviews.On("Approve", mock.Anything, reviewID, staffID, "1234").Return(paymentsmodels.StaffReviewResult{
		ReviewID:      reviewID,
		ReferenceID:   referenceID,
		ReferenceType: paymentsmodels.ReferenceTypeOrder,
		Action:        "approve",
	}, nil)

	e := echo.New()
	e.GET("/payments/staff/reviews", func(c echo.Context) error {
		ctx := context.WithValue(c.Request().Context(), models.SessionKey, models.ActiveSession{UserID: staffID})
		c.SetRequest(c.Request().WithContext(ctx))
		return h.HandleWebSocket(c)
	})
	server := httptest.NewServer(e)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/payments/staff/reviews"
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
	defer func() { _ = conn.Close() }()

	_, connected, err := conn.ReadMessage()
	require.NoError(t, err)
	require.JSONEq(t, `{"type":"connected"}`, string(connected))

	require.NoError(t, conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"approve","review_id":"`+reviewID.String()+`","reference_last4":"1234"}`)))

	_, okMsg, err := conn.ReadMessage()
	require.NoError(t, err)
	require.Contains(t, string(okMsg), `"type":"action_ok"`)
	require.Contains(t, string(okMsg), `"reference_type":"order"`)
	require.Contains(t, string(okMsg), referenceID.String())
	require.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)

	require.NoError(t, conn.Close())
	select {
	case <-unregistered:
	case <-time.After(2 * time.Second):
		t.Fatal("Unregister no fue llamado al cerrar el websocket")
	}
}
