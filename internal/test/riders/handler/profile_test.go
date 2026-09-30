package handler_test

import (
	"encoding/json"
	"net/http"
	"testing"

	mocksriders "Encargalo.app-api.go/internal/mocks/riders/domain/ports"
	"Encargalo.app-api.go/internal/riders/domain/dtos"
	ridershandler "Encargalo.app-api.go/internal/riders/handler"
	"Encargalo.app-api.go/internal/shared/buildhttptest"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRiderGetProfile_Success(t *testing.T) {
	svc := mocksriders.NewRiderProfileApp(t)
	h := ridershandler.NewRiderProfileHandler(svc)
	riderID := uuid.New()
	plate := "ABCD123"

	tc := buildhttptest.SetupControllerCase("/riders/profile", http.MethodGet, nil)
	tc = withRiderSession(tc, riderID)

	svc.On("GetOwnProfile", tc.Req.Context(), riderID).Return(dtos.RiderSelfProfileResponse{
		Driver:  dtos.RiderSelfDriver{UserID: riderID.String(), Name: "Ada", SurName: "Lovelace"},
		Vehicle: dtos.RiderSelfVehicle{Plate: &plate},
	}, nil)

	err := h.GetProfile(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
	var body struct {
		Driver  map[string]any `json:"driver"`
		Vehicle map[string]any `json:"vehicle"`
	}
	require.NoError(t, json.Unmarshal(tc.Res.Body.Bytes(), &body))
	assert.Equal(t, "Ada", body.Driver["name"])
	assert.Equal(t, "ABCD123", body.Vehicle["plate"])
}

func TestRiderGetProfile_EmptyVehicleSerializesAsObject(t *testing.T) {
	svc := mocksriders.NewRiderProfileApp(t)
	h := ridershandler.NewRiderProfileHandler(svc)
	riderID := uuid.New()

	tc := buildhttptest.SetupControllerCase("/riders/profile", http.MethodGet, nil)
	tc = withRiderSession(tc, riderID)

	svc.On("GetOwnProfile", tc.Req.Context(), riderID).Return(dtos.RiderSelfProfileResponse{
		Driver: dtos.RiderSelfDriver{UserID: riderID.String(), Name: "Ada"},
	}, nil)

	require.NoError(t, h.GetProfile(tc.Ctx))

	var body struct {
		Vehicle map[string]any `json:"vehicle"`
	}
	require.NoError(t, json.Unmarshal(tc.Res.Body.Bytes(), &body))
	assert.NotNil(t, body.Vehicle)
	assert.Empty(t, body.Vehicle)
}

func TestRiderGetProfile_ServiceErrorMapsToHTTPError(t *testing.T) {
	svc := mocksriders.NewRiderProfileApp(t)
	h := ridershandler.NewRiderProfileHandler(svc)
	riderID := uuid.New()

	tc := buildhttptest.SetupControllerCase("/riders/profile", http.MethodGet, nil)
	tc = withRiderSession(tc, riderID)

	svc.On("GetOwnProfile", tc.Req.Context(), riderID).Return(dtos.RiderSelfProfileResponse{}, errcustom.ErrNotFound)

	err := h.GetProfile(tc.Ctx)

	assert.Error(t, err)
}
