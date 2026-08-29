package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"

	authModels "Encargalo.app-api.go/internal/auth/domain/models"
	mockstrips "Encargalo.app-api.go/internal/mocks/trips/domain/ports"
	"Encargalo.app-api.go/internal/shared/buildhttptest"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/trips/domain/dtos"
	tripshandler "Encargalo.app-api.go/internal/trips/handler"
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

func newTripsHandler(t *testing.T) (tripshandler.TripsHandler, *mockstrips.TripsApp) {
	t.Helper()
	svc := mockstrips.NewTripsApp(t)
	return tripshandler.NewTripsHandler(svc), svc
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

func sampleCreateTripRequest() dtos.CreateTripRequest {
	return dtos.CreateTripRequest{
		PickupAddress:    "Av. Libertador, Caracas",
		PickupLatitude:   10.4806,
		PickupLongitude:  -66.9036,
		DropoffAddress:   "CC Sambil, Caracas",
		DropoffLatitude:  10.4956,
		DropoffLongitude: -66.8517,
	}
}

func TestRequestTrip_InvalidPayload(t *testing.T) {
	h, _ := newTripsHandler(t)
	tc := buildhttptest.SetupControllerCase("/trips", http.MethodPost, buildBody(t, map[string]string{"pickup_address": ""}))
	tc = withSession(tc, authModels.ActiveSession{UserID: uuid.New()})

	err := h.RequestTrip(tc.Ctx)

	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestRequestTrip_Success(t *testing.T) {
	h, svc := newTripsHandler(t)
	customerID := uuid.New()
	req := sampleCreateTripRequest()
	tc := buildhttptest.SetupControllerCase("/trips", http.MethodPost, buildBody(t, req))
	tc = withSession(tc, authModels.ActiveSession{UserID: customerID})

	svc.On("RequestTrip", tc.Req.Context(), customerID, req).Return(dtos.TripResponse{
		ID: uuid.New(), CustomerID: customerID, Status: "requested",
	}, nil)

	err := h.RequestTrip(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, tc.Res.Code)
}

func TestGetTrip_InvalidID(t *testing.T) {
	h, _ := newTripsHandler(t)
	tc := buildhttptest.SetupControllerCase("/trips/not-a-uuid", http.MethodGet, nil)
	tc = withSession(tc, authModels.ActiveSession{UserID: uuid.New()})
	tc.Ctx.SetParamNames("id")
	tc.Ctx.SetParamValues("not-a-uuid")

	err := h.GetTrip(tc.Ctx)

	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestGetTrip_ForbiddenMapsTo403(t *testing.T) {
	h, svc := newTripsHandler(t)
	tripID, customerID := uuid.New(), uuid.New()
	tc := buildhttptest.SetupControllerCase("/trips/"+tripID.String(), http.MethodGet, nil)
	tc = withSession(tc, authModels.ActiveSession{UserID: customerID})
	tc.Ctx.SetParamNames("id")
	tc.Ctx.SetParamValues(tripID.String())

	svc.On("GetTrip", tc.Req.Context(), tripID, customerID).Return(dtos.TripResponse{}, errcustom.ErrTripForbidden)

	err := h.GetTrip(tc.Ctx)

	assert.Equal(t, http.StatusForbidden, httpCode(err))
}

func TestGetTripHistory_Success(t *testing.T) {
	h, svc := newTripsHandler(t)
	customerID := uuid.New()
	tc := buildhttptest.SetupControllerCase("/trips/history", http.MethodGet, nil)
	tc = withSession(tc, authModels.ActiveSession{UserID: customerID})

	svc.On("GetTripHistory", tc.Req.Context(), customerID).Return(dtos.TripHistoryResponse{
		Trips: []dtos.TripResponse{{ID: uuid.New(), CustomerID: customerID}},
	}, nil)

	err := h.GetTripHistory(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
}

func TestCancelTrip_NotCancellableMapsTo422(t *testing.T) {
	h, svc := newTripsHandler(t)
	tripID, customerID := uuid.New(), uuid.New()
	tc := buildhttptest.SetupControllerCase("/trips/"+tripID.String()+"/cancel", http.MethodPost, nil)
	tc = withSession(tc, authModels.ActiveSession{UserID: customerID})
	tc.Ctx.SetParamNames("id")
	tc.Ctx.SetParamValues(tripID.String())

	svc.On("CancelTrip", tc.Req.Context(), tripID, customerID).Return(dtos.TripResponse{}, errcustom.ErrTripNotCancellable)

	err := h.CancelTrip(tc.Ctx)

	assert.Equal(t, http.StatusUnprocessableEntity, httpCode(err))
}

func TestCancelTrip_Success(t *testing.T) {
	h, svc := newTripsHandler(t)
	tripID, customerID := uuid.New(), uuid.New()
	tc := buildhttptest.SetupControllerCase("/trips/"+tripID.String()+"/cancel", http.MethodPost, nil)
	tc = withSession(tc, authModels.ActiveSession{UserID: customerID})
	tc.Ctx.SetParamNames("id")
	tc.Ctx.SetParamValues(tripID.String())

	svc.On("CancelTrip", tc.Req.Context(), tripID, customerID).Return(dtos.TripResponse{
		ID: tripID, CustomerID: customerID, Status: "cancelled",
	}, nil)

	err := h.CancelTrip(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
}
