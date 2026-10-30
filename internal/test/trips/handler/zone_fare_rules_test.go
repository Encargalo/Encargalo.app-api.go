package handler_test

import (
	"net/http"
	"testing"

	mockstrips "Encargalo.app-api.go/internal/mocks/trips/domain/ports"
	"Encargalo.app-api.go/internal/shared/buildhttptest"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/trips/domain/dtos"
	tripshandler "Encargalo.app-api.go/internal/trips/handler"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func newStaffZoneFareRulesHandler(t *testing.T) (tripshandler.StaffZoneFareRulesHandler, *mockstrips.ZoneFareRulesApp) {
	t.Helper()
	svc := mockstrips.NewZoneFareRulesApp(t)
	return tripshandler.NewStaffZoneFareRulesHandler(svc), svc
}

func sampleZoneFareRuleRequest() dtos.ZoneFareRuleRequest {
	return dtos.ZoneFareRuleRequest{
		Name:                          "Charallave",
		IncludedKm:                    3,
		DayMinFareUSD:                 1.2375,
		DayExtraRatePerKmUSD:          0.30,
		NightMinFareUSD:               1.575,
		NightExtraRatePerKmUSD:        0.50,
		EarlyMorningMinFareUSD:        1.80,
		EarlyMorningExtraRatePerKmUSD: 0.70,
		Polygon: []dtos.LatLngPoint{
			{Lat: 10.12, Lng: -66.85},
			{Lat: 10.13, Lng: -66.85},
			{Lat: 10.13, Lng: -66.84},
		},
	}
}

func TestCreateZoneFareRule_InvalidPayload(t *testing.T) {
	h, _ := newStaffZoneFareRulesHandler(t)
	tc := buildhttptest.SetupControllerCase("/trips/staff/zone-fare-rules", http.MethodPost, buildBody(t, map[string]string{}))

	err := h.CreateZoneFareRule(tc.Ctx)

	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestCreateZoneFareRule_Success(t *testing.T) {
	h, svc := newStaffZoneFareRulesHandler(t)
	req := sampleZoneFareRuleRequest()
	tc := buildhttptest.SetupControllerCase("/trips/staff/zone-fare-rules", http.MethodPost, buildBody(t, req))

	svc.On("CreateZoneFareRule", tc.Req.Context(), req).Return(dtos.ZoneFareRuleResponse{ZoneID: uuid.New(), Name: req.Name}, nil)

	err := h.CreateZoneFareRule(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, tc.Res.Code)
}

func TestListZoneFareRules_Success(t *testing.T) {
	h, svc := newStaffZoneFareRulesHandler(t)
	tc := buildhttptest.SetupControllerCase("/trips/staff/zone-fare-rules", http.MethodGet, nil)

	svc.On("ListZoneFareRules", tc.Req.Context()).Return(dtos.ZoneFareRulesResponse{Zones: []dtos.ZoneFareRuleResponse{{ZoneID: uuid.New()}}}, nil)

	err := h.ListZoneFareRules(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
}

func TestGetZoneFareRule_InvalidID(t *testing.T) {
	h, _ := newStaffZoneFareRulesHandler(t)
	tc := buildhttptest.SetupControllerCase("/trips/staff/zone-fare-rules/not-a-uuid", http.MethodGet, nil)
	tc.Ctx.SetParamNames("zone_id")
	tc.Ctx.SetParamValues("not-a-uuid")

	err := h.GetZoneFareRule(tc.Ctx)

	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestGetZoneFareRule_NotFound(t *testing.T) {
	h, svc := newStaffZoneFareRulesHandler(t)
	zoneID := uuid.New()
	tc := buildhttptest.SetupControllerCase("/trips/staff/zone-fare-rules/"+zoneID.String(), http.MethodGet, nil)
	tc.Ctx.SetParamNames("zone_id")
	tc.Ctx.SetParamValues(zoneID.String())

	svc.On("GetZoneFareRule", tc.Req.Context(), zoneID).Return(dtos.ZoneFareRuleResponse{}, errcustom.ErrNotFound)

	err := h.GetZoneFareRule(tc.Ctx)

	assert.Equal(t, http.StatusNotFound, httpCode(err))
}

func TestUpdateZoneFareRule_Success(t *testing.T) {
	h, svc := newStaffZoneFareRulesHandler(t)
	zoneID := uuid.New()
	req := sampleZoneFareRuleRequest()
	tc := buildhttptest.SetupControllerCase("/trips/staff/zone-fare-rules/"+zoneID.String(), http.MethodPatch, buildBody(t, req))
	tc.Ctx.SetParamNames("zone_id")
	tc.Ctx.SetParamValues(zoneID.String())

	svc.On("UpdateZoneFareRule", tc.Req.Context(), zoneID, req).Return(dtos.ZoneFareRuleResponse{ZoneID: zoneID, Name: req.Name}, nil)

	err := h.UpdateZoneFareRule(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
}

func TestDeactivateZoneFareRule_Success(t *testing.T) {
	h, svc := newStaffZoneFareRulesHandler(t)
	zoneID := uuid.New()
	tc := buildhttptest.SetupControllerCase("/trips/staff/zone-fare-rules/"+zoneID.String(), http.MethodDelete, nil)
	tc.Ctx.SetParamNames("zone_id")
	tc.Ctx.SetParamValues(zoneID.String())

	svc.On("DeactivateZoneFareRule", tc.Req.Context(), zoneID).Return(nil)

	err := h.DeactivateZoneFareRule(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, tc.Res.Code)
}
