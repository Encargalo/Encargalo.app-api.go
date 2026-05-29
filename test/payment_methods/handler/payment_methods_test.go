package handler_test

import (
	"encoding/json"
	"net/http"
	"testing"

	mockspm "Encargalo.app-api.go/internal/mocks/payment_methods/domain/ports"
	pmmodels "Encargalo.app-api.go/internal/payment_methods/domain/models"
	pmhandler "Encargalo.app-api.go/internal/payment_methods/handler"
	"Encargalo.app-api.go/internal/shared/buildhttptest"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func newHandler(t *testing.T) (pmhandler.PaymentMethods, *mockspm.PaymentMethodsApp) {
	t.Helper()
	svc := mockspm.NewPaymentMethodsApp(t)
	return pmhandler.NewPaymentMethodsHandler(svc), svc
}

func TestGetPaymentMethods_Success(t *testing.T) {
	h, svc := newHandler(t)

	fields := json.RawMessage(`[]`)
	qrURL := "https://res.cloudinary.com/example/qr.jpg"

	m1 := pmmodels.PaymentMethod{Type: "EfectivoBs", Name: "Efectivo en Bolívares", Description: "Bs", Fields: fields, IsActive: true}
	m1.ID = uuid.New()
	m2 := pmmodels.PaymentMethod{Type: "EfectivoDolar", Name: "Efectivo en Dólares", Description: "USD", Fields: fields, IsActive: true}
	m2.ID = uuid.New()
	m3 := pmmodels.PaymentMethod{Type: "PagoMovil", Name: "Pago Móvil", Description: "Interbancario", Fields: json.RawMessage(`[{"key":"nombre","label":"Nombre","required":true}]`), QRUrl: &qrURL, IsActive: true}
	m3.ID = uuid.New()
	methods := []pmmodels.PaymentMethod{m1, m2, m3}

	tc := buildhttptest.SetupControllerCase("/payment-methods", http.MethodGet, nil)
	svc.On("GetActivePaymentMethods", tc.Req.Context()).Return(methods, nil)

	err := h.GetPaymentMethods(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)

	var result []pmmodels.PaymentMethod
	assert.NoError(t, json.Unmarshal(tc.Res.Body.Bytes(), &result))
	assert.Len(t, result, 3)
	assert.Equal(t, "EfectivoBs", result[0].Type)
	assert.Equal(t, "PagoMovil", result[2].Type)
	assert.NotNil(t, result[2].QRUrl)
}

func TestGetPaymentMethods_Empty(t *testing.T) {
	h, svc := newHandler(t)

	tc := buildhttptest.SetupControllerCase("/payment-methods", http.MethodGet, nil)
	svc.On("GetActivePaymentMethods", tc.Req.Context()).Return([]pmmodels.PaymentMethod{}, nil)

	err := h.GetPaymentMethods(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)

	var result []pmmodels.PaymentMethod
	assert.NoError(t, json.Unmarshal(tc.Res.Body.Bytes(), &result))
	assert.Empty(t, result)
}

func TestGetPaymentMethods_AppError(t *testing.T) {
	h, svc := newHandler(t)

	tc := buildhttptest.SetupControllerCase("/payment-methods", http.MethodGet, nil)
	svc.On("GetActivePaymentMethods", tc.Req.Context()).Return(nil, assert.AnError)

	err := h.GetPaymentMethods(tc.Ctx)

	assert.Error(t, err)
}
