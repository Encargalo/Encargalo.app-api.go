package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	authmodels "Encargalo.app-api.go/internal/auth/domain/models"
	mocksemployee "Encargalo.app-api.go/internal/mocks/employee/domain/ports"
	mocksorders "Encargalo.app-api.go/internal/mocks/orders/domain/ports"
	"Encargalo.app-api.go/internal/orders/domain/dtos"
	ordershandler "Encargalo.app-api.go/internal/orders/handler"
	"Encargalo.app-api.go/internal/shared/buildhttptest"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newOrderHandler(t *testing.T) (ordershandler.OrderHandler, *mocksorders.OrdersApp, *mocksorders.AppManagerOrder) {
	t.Helper()
	svc := mocksorders.NewOrdersApp(t)
	mgr := mocksorders.NewAppManagerOrder(t)
	employee := mocksemployee.NewEmployeeApp(t)
	return ordershandler.NewOrderHandler(svc, mgr, employee), svc, mgr
}

func withSession(tc buildhttptest.ControllerCase, userID uuid.UUID) buildhttptest.ControllerCase {
	session := authmodels.ActiveSession{ID: uuid.New(), UserID: userID}
	ctx := context.WithValue(tc.Req.Context(), authmodels.SessionKey, session)
	tc.Req = tc.Req.WithContext(ctx)
	tc.Ctx.SetRequest(tc.Req)
	return tc
}

func TestGetOrderHistory_Success(t *testing.T) {
	h, svc, _ := newOrderHandler(t)

	customerID := uuid.New()
	shopID := uuid.New()
	expected := dtos.OrderHistoryResponse{
		{
			ID:            uuid.New(),
			Number:        1,
			Status:        "Completed",
			TotalPrice:    25.50,
			DeliveryFee:   2.00,
			MethodPayment: "cash",
			Address:       "Av. Principal 123",
			ShopID:        shopID,
			CreatedAt:     time.Now(),
		},
	}

	svc.On("GetOrderHistory", mock.Anything, customerID).Return(expected, nil)

	tc := buildhttptest.SetupControllerCase("/orders/history", http.MethodGet, nil)
	tc = withSession(tc, customerID)

	err := h.GetOrderHistory(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)

	var got dtos.OrderHistoryResponse
	assert.NoError(t, json.NewDecoder(tc.Res.Body).Decode(&got))
	assert.Len(t, got, 1)
	assert.Equal(t, expected[0].Status, got[0].Status)
}

func TestGetOrderHistory_EmptyList(t *testing.T) {
	h, svc, _ := newOrderHandler(t)

	customerID := uuid.New()
	svc.On("GetOrderHistory", mock.Anything, customerID).Return(dtos.OrderHistoryResponse{}, nil)

	tc := buildhttptest.SetupControllerCase("/orders/history", http.MethodGet, nil)
	tc = withSession(tc, customerID)

	err := h.GetOrderHistory(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)

	var got dtos.OrderHistoryResponse
	assert.NoError(t, json.NewDecoder(tc.Res.Body).Decode(&got))
	assert.Empty(t, got)
}

func TestGetOrderHistory_NoSession(t *testing.T) {
	h, _, _ := newOrderHandler(t)

	tc := buildhttptest.SetupControllerCase("/orders/history", http.MethodGet, nil)

	err := h.GetOrderHistory(tc.Ctx)

	assert.ErrorIs(t, err, errcustom.ErrUnexpectedError)
}

func TestGetOrderHistory_RepoError(t *testing.T) {
	h, svc, _ := newOrderHandler(t)

	customerID := uuid.New()
	svc.On("GetOrderHistory", mock.Anything, customerID).Return(nil, errcustom.ErrUnexpectedError)

	tc := buildhttptest.SetupControllerCase("/orders/history", http.MethodGet, nil)
	tc = withSession(tc, customerID)

	err := h.GetOrderHistory(tc.Ctx)

	assert.ErrorIs(t, err, errcustom.ErrUnexpectedError)
}
