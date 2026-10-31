package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"

	authModels "Encargalo.app-api.go/internal/auth/domain/models"
	mocksshipments "Encargalo.app-api.go/internal/mocks/shipments/domain/ports"
	"Encargalo.app-api.go/internal/shared/buildhttptest"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/shipments/domain/dtos"
	shipmentshandler "Encargalo.app-api.go/internal/shipments/handler"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func httpCode(err error) int {
	if he, ok := err.(*echo.HTTPError); ok {
		return he.Code
	}
	return 0
}

func newShipmentsHandler(t *testing.T) (shipmentshandler.ShipmentsHandler, *mocksshipments.ShipmentsApp) {
	t.Helper()
	svc := mocksshipments.NewShipmentsApp(t)
	return shipmentshandler.NewShipmentsHandler(svc, nil), svc
}

func buildBody(t *testing.T, v any) *bytes.Buffer {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return bytes.NewBuffer(b)
}

func withSession(tc buildhttptest.ControllerCase, session authModels.ActiveSession) buildhttptest.ControllerCase {
	ctx := context.WithValue(tc.Req.Context(), authModels.SessionKey, session)
	tc.Req = tc.Req.WithContext(ctx)
	tc.Ctx.SetRequest(tc.Req)
	return tc
}

func sampleCreateShipmentRequest() dtos.CreateShipmentRequest {
	return dtos.CreateShipmentRequest{
		QuoteID:            uuid.New(),
		MethodPayment:      "Balance",
		Sender:             dtos.ShipmentContact{IsCustomer: true},
		Recipient:          dtos.ShipmentContact{IsCustomer: false, Name: "Juan Pérez", Phone: "+584121234567"},
		PackageDescription: "Una torta de cumpleaños",
	}
}

func TestRequestShipment_InvalidPayload(t *testing.T) {
	h, _ := newShipmentsHandler(t)
	tc := buildhttptest.SetupControllerCase("/shipments", http.MethodPost, buildBody(t, map[string]string{}))
	tc = withSession(tc, authModels.ActiveSession{UserID: uuid.New()})

	err := h.RequestShipment(tc.Ctx)

	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestRequestShipment_ThirdPartyWithoutContactIsInvalid(t *testing.T) {
	h, _ := newShipmentsHandler(t)
	req := sampleCreateShipmentRequest()
	req.Recipient = dtos.ShipmentContact{IsCustomer: false}
	tc := buildhttptest.SetupControllerCase("/shipments", http.MethodPost, buildBody(t, req))
	tc = withSession(tc, authModels.ActiveSession{UserID: uuid.New()})

	err := h.RequestShipment(tc.Ctx)

	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestRequestShipment_Success(t *testing.T) {
	h, svc := newShipmentsHandler(t)
	customerID := uuid.New()
	req := sampleCreateShipmentRequest()
	tc := buildhttptest.SetupControllerCase("/shipments", http.MethodPost, buildBody(t, req))
	tc = withSession(tc, authModels.ActiveSession{UserID: customerID})

	svc.On("RequestShipment", tc.Req.Context(), customerID, req).Return(dtos.ShipmentResponse{
		ID: uuid.New(), CustomerID: customerID, Status: "searching",
		PickupCode: "1234", DropoffCode: "5678",
	}, nil)

	err := h.RequestShipment(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, tc.Res.Code)
}

func TestEstimateFare_Success(t *testing.T) {
	h, svc := newShipmentsHandler(t)
	customerID := uuid.New()
	body := dtos.FareEstimateRequest{
		PickupAddress:    "Av. Libertador, Caracas",
		PickupLatitude:   10.4806,
		PickupLongitude:  -66.9036,
		DropoffAddress:   "CC Sambil, Caracas",
		DropoffLatitude:  10.4956,
		DropoffLongitude: -66.8517,
	}
	tc := buildhttptest.SetupControllerCase("/shipments/fare-estimate", http.MethodPost, buildBody(t, body))
	tc = withSession(tc, authModels.ActiveSession{UserID: customerID})

	expectedReq := body
	expectedReq.CustomerID = customerID
	svc.On("EstimateFare", tc.Req.Context(), expectedReq).Return(dtos.FareEstimateResponse{
		QuoteID: uuid.New(), DistanceMeters: 1500, DurationMin: 6, FareUSD: 0.8, FareBs: 32,
	}, nil)

	err := h.EstimateFare(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
}

// TestGetShipment_CodesAlwaysPresent es la regresión más importante de este
// dominio frente a trips: pickup_code/dropoff_code deben salir no vacíos en
// la respuesta sin importar el estado del envío.
func TestGetShipment_CodesAlwaysPresent(t *testing.T) {
	statuses := []string{"requested", "payment_review", "searching", "accepted", "in_transit", "delivered", "cancelled", "no_driver"}

	for _, status := range statuses {
		t.Run(status, func(t *testing.T) {
			h, svc := newShipmentsHandler(t)
			shipmentID, customerID := uuid.New(), uuid.New()
			tc := buildhttptest.SetupControllerCase("/shipments/"+shipmentID.String(), http.MethodGet, nil)
			tc = withSession(tc, authModels.ActiveSession{UserID: customerID})
			tc.Ctx.SetParamNames("id")
			tc.Ctx.SetParamValues(shipmentID.String())

			svc.On("GetShipment", tc.Req.Context(), shipmentID, customerID).Return(dtos.ShipmentResponse{
				ID: shipmentID, CustomerID: customerID, Status: status,
				PickupCode: "1234", DropoffCode: "5678",
			}, nil)

			err := h.GetShipment(tc.Ctx)

			assert.NoError(t, err)
			var body map[string]any
			assert.NoError(t, json.Unmarshal(tc.Res.Body.Bytes(), &body))
			assert.Equal(t, "1234", body["pickup_code"])
			assert.Equal(t, "5678", body["dropoff_code"])
		})
	}
}

func TestGetShipment_ForbiddenMapsTo403(t *testing.T) {
	h, svc := newShipmentsHandler(t)
	shipmentID, customerID := uuid.New(), uuid.New()
	tc := buildhttptest.SetupControllerCase("/shipments/"+shipmentID.String(), http.MethodGet, nil)
	tc = withSession(tc, authModels.ActiveSession{UserID: customerID})
	tc.Ctx.SetParamNames("id")
	tc.Ctx.SetParamValues(shipmentID.String())

	svc.On("GetShipment", tc.Req.Context(), shipmentID, customerID).Return(dtos.ShipmentResponse{}, errcustom.ErrShipmentForbidden)

	err := h.GetShipment(tc.Ctx)

	assert.Equal(t, http.StatusForbidden, httpCode(err))
}

func TestGetShipmentHistory_Success(t *testing.T) {
	h, svc := newShipmentsHandler(t)
	customerID := uuid.New()
	tc := buildhttptest.SetupControllerCase("/shipments/history", http.MethodGet, nil)
	tc = withSession(tc, authModels.ActiveSession{UserID: customerID})

	svc.On("GetShipmentHistory", tc.Req.Context(), customerID).Return(dtos.ShipmentHistoryResponse{
		Shipments: []dtos.ShipmentResponse{{ID: uuid.New(), CustomerID: customerID}},
	}, nil)

	err := h.GetShipmentHistory(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
}

func TestCancelShipment_NotCancellableMapsTo422(t *testing.T) {
	h, svc := newShipmentsHandler(t)
	shipmentID, customerID := uuid.New(), uuid.New()
	tc := buildhttptest.SetupControllerCase("/shipments/"+shipmentID.String()+"/cancel", http.MethodPost, nil)
	tc = withSession(tc, authModels.ActiveSession{UserID: customerID})
	tc.Ctx.SetParamNames("id")
	tc.Ctx.SetParamValues(shipmentID.String())

	svc.On("CancelShipment", tc.Req.Context(), shipmentID, customerID).Return(dtos.ShipmentResponse{}, errcustom.ErrShipmentNotCancellable)

	err := h.CancelShipment(tc.Ctx)

	assert.Equal(t, http.StatusUnprocessableEntity, httpCode(err))
}

func TestCancelShipment_Success(t *testing.T) {
	h, svc := newShipmentsHandler(t)
	shipmentID, customerID := uuid.New(), uuid.New()
	tc := buildhttptest.SetupControllerCase("/shipments/"+shipmentID.String()+"/cancel", http.MethodPost, nil)
	tc = withSession(tc, authModels.ActiveSession{UserID: customerID})
	tc.Ctx.SetParamNames("id")
	tc.Ctx.SetParamValues(shipmentID.String())

	svc.On("CancelShipment", tc.Req.Context(), shipmentID, customerID).Return(dtos.ShipmentResponse{
		ID: shipmentID, CustomerID: customerID, Status: "cancelled",
	}, nil)

	err := h.CancelShipment(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
}
