package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"

	authmodels "Encargalo.app-api.go/internal/auth/domain/models"
	mocksorders "Encargalo.app-api.go/internal/mocks/orders/domain/ports"
	"Encargalo.app-api.go/internal/orders/domain/models"
	ordershandler "Encargalo.app-api.go/internal/orders/handler"
	"Encargalo.app-api.go/internal/shared/buildhttptest"
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

func newHandler(t *testing.T) (ordershandler.OrderHandler, *mocksorders.OrdersApp, *mocksorders.AppManagerOrder) {
	t.Helper()
	svc := mocksorders.NewOrdersApp(t)
	mgr := mocksorders.NewAppManagerOrder(t)
	return ordershandler.NewOrderHandler(svc, mgr), svc, mgr
}

func withSession(tc buildhttptest.ControllerCase, session authmodels.ActiveSession) buildhttptest.ControllerCase {
	ctx := context.WithValue(tc.Req.Context(), authmodels.SessionKey, session)
	tc.Req = tc.Req.WithContext(ctx)
	tc.Ctx.SetRequest(tc.Req)
	return tc
}

func session() authmodels.ActiveSession {
	return authmodels.ActiveSession{UserID: uuid.New()}
}

// --- CreateOrder: Efectivo ---

func TestCreateOrder_InvalidMethod_EfectivoBs(t *testing.T) {
	h, _, _ := newHandler(t)

	body := buildBody(t, map[string]any{
		"id":             uuid.New().String(),
		"shop_id":        uuid.New().String(),
		"method_payment": "EfectivoBs",
		"delivery_fee":   5000,
		"address": map[string]any{
			"address":   "Av. Principal",
			"latitude":  10.48,
			"longitude": -66.87,
		},
		"items": []map[string]any{
			{"item_id": uuid.New().String(), "amount": 1, "observation": "", "flavors": map[string]any{}, "additions": []any{}},
		},
	})

	tc := buildhttptest.SetupControllerCase("/orders", http.MethodPost, body)
	tc = withSession(tc, session())

	err := h.CreateOrder(tc.Ctx)

	assert.Error(t, err)
	he, ok := err.(*echo.HTTPError)
	if assert.True(t, ok) {
		assert.Equal(t, http.StatusBadRequest, he.Code)
	}
}

func TestCreateOrder_InvalidMethod_EfectivoDolar(t *testing.T) {
	h, _, _ := newHandler(t)

	body := buildBody(t, map[string]any{
		"id":             uuid.New().String(),
		"shop_id":        uuid.New().String(),
		"method_payment": "EfectivoDolar",
		"delivery_fee":   5000,
		"address": map[string]any{
			"address":   "Av. Principal",
			"latitude":  10.48,
			"longitude": -66.87,
		},
		"items": []map[string]any{
			{"item_id": uuid.New().String(), "amount": 1, "observation": "", "flavors": map[string]any{}, "additions": []any{}},
		},
	})

	tc := buildhttptest.SetupControllerCase("/orders", http.MethodPost, body)
	tc = withSession(tc, session())

	err := h.CreateOrder(tc.Ctx)

	assert.Error(t, err)
	he, ok := err.(*echo.HTTPError)
	if assert.True(t, ok) {
		assert.Equal(t, http.StatusBadRequest, he.Code)
	}
}

func TestCreateOrder_Success_PagoMovil(t *testing.T) {
	h, svc, mgr := newHandler(t)

	body := buildBody(t, map[string]any{
		"id":             uuid.New().String(),
		"shop_id":        uuid.New().String(),
		"method_payment": "PagoMovil",
		"pago_movil_data": map[string]any{
			"nombre":   "Juan Pérez",
			"cedula":   "V-12345678",
			"telefono": "0412-1234567",
			"banco":    "Banco de Venezuela",
		},
		"payment_proof_image":  "data:image/jpeg;base64,/9j/abc123",
		"delivery_fee":         5000,
		"expected_total_price": 18.5,
		"address": map[string]any{
			"address":   "Av. Principal",
			"latitude":  10.48,
			"longitude": -66.87,
		},
		"items": []map[string]any{
			{"item_id": uuid.New().String(), "amount": 1, "observation": "", "flavors": map[string]any{}, "additions": []any{}},
		},
	})

	tc := buildhttptest.SetupControllerCase("/orders", http.MethodPost, body)
	tc = withSession(tc, session())

	order := &models.Order{ID: uuid.New(), Status: string(models.StatusPaymentReview)}
	svc.On("CreateOrder", mock.Anything, mock.Anything).Return(order, nil)
	mgr.On("BroadcastOrderToCustomer", *order).Return(nil)

	err := h.CreateOrder(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, tc.Res.Code)
}

// --- CreateOrder: sin sesión ---

func TestCreateOrder_NoSession(t *testing.T) {
	h, _, _ := newHandler(t)

	body := buildBody(t, map[string]any{
		"id":             uuid.New().String(),
		"shop_id":        uuid.New().String(),
		"method_payment": "EfectivoBs",
		"delivery_fee":   5000,
		"address":        map[string]any{"address": "x", "latitude": 1.0, "longitude": 1.0},
		"items":          []any{},
	})

	tc := buildhttptest.SetupControllerCase("/orders", http.MethodPost, body)
	// sin withSession

	err := h.CreateOrder(tc.Ctx)

	assert.Error(t, err)
}

// --- CreateOrder: método de pago inválido ---

func TestCreateOrder_InvalidPaymentMethod(t *testing.T) {
	h, _, _ := newHandler(t)

	body := buildBody(t, map[string]any{
		"id":             uuid.New().String(),
		"shop_id":        uuid.New().String(),
		"method_payment": "Nequi",
		"delivery_fee":   5000,
		"address":        map[string]any{"address": "x", "latitude": 1.0, "longitude": 1.0},
		"items": []map[string]any{
			{"item_id": uuid.New().String(), "amount": 1, "observation": "", "flavors": map[string]any{}, "additions": []any{}},
		},
	})

	tc := buildhttptest.SetupControllerCase("/orders", http.MethodPost, body)
	tc = withSession(tc, session())

	err := h.CreateOrder(tc.Ctx)

	assert.Error(t, err)
	he, ok := err.(*echo.HTTPError)
	if assert.True(t, ok) {
		assert.Equal(t, http.StatusBadRequest, he.Code)
	}
}

// --- CreateOrder: PagoMovil sin pago_movil_data ---

func TestCreateOrder_Success_PagoMovil_OnlyProof(t *testing.T) {
	h, svc, mgr := newHandler(t)

	body := buildBody(t, map[string]any{
		"id":                   uuid.New().String(),
		"shop_id":              uuid.New().String(),
		"method_payment":       "PagoMovil",
		"payment_proof_image":  "data:image/jpeg;base64,/9j/abc",
		"delivery_fee":         5000,
		"expected_total_price": 18.5,
		"address":              map[string]any{"address": "x", "latitude": 1.0, "longitude": 1.0},
		"items": []map[string]any{
			{"item_id": uuid.New().String(), "amount": 1, "observation": "", "flavors": map[string]any{}, "additions": []any{}},
		},
	})

	tc := buildhttptest.SetupControllerCase("/orders", http.MethodPost, body)
	tc = withSession(tc, session())

	order := &models.Order{ID: uuid.New(), Status: string(models.StatusPaymentReview)}
	svc.On("CreateOrder", mock.Anything, mock.Anything).Return(order, nil)
	mgr.On("BroadcastOrderToCustomer", *order).Return(nil)

	err := h.CreateOrder(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, tc.Res.Code)
}

// --- CreateOrder: PagoMovil sin comprobante ---

func TestCreateOrder_PagoMovil_MissingProofImage(t *testing.T) {
	h, _, _ := newHandler(t)

	body := buildBody(t, map[string]any{
		"id":             uuid.New().String(),
		"shop_id":        uuid.New().String(),
		"method_payment": "PagoMovil",
		"pago_movil_data": map[string]any{
			"nombre":   "Juan Pérez",
			"cedula":   "V-12345678",
			"telefono": "0412-1234567",
			"banco":    "Banco de Venezuela",
		},
		// sin payment_proof_image
		"delivery_fee": 5000,
		"address":      map[string]any{"address": "x", "latitude": 1.0, "longitude": 1.0},
		"items": []map[string]any{
			{"item_id": uuid.New().String(), "amount": 1, "observation": "", "flavors": map[string]any{}, "additions": []any{}},
		},
	})

	tc := buildhttptest.SetupControllerCase("/orders", http.MethodPost, body)
	tc = withSession(tc, session())

	err := h.CreateOrder(tc.Ctx)

	assert.Error(t, err)
	he, ok := err.(*echo.HTTPError)
	if assert.True(t, ok) {
		assert.Equal(t, http.StatusBadRequest, he.Code)
	}
}
