package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	mocksriders "Encargalo.app-api.go/internal/mocks/riders/domain/ports"
	"Encargalo.app-api.go/internal/riders/domain/dtos"
	ridershandler "Encargalo.app-api.go/internal/riders/handler"
	"Encargalo.app-api.go/internal/shared/buildhttptest"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func buildBody(t *testing.T, v any) *bytes.Buffer {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return bytes.NewBuffer(b)
}

func TestGetApplication_Success(t *testing.T) {
	svc := mocksriders.NewRiderProfileApp(t)
	h := ridershandler.NewRiderApplicationHandler(svc)
	userID := uuid.New()

	tc := buildhttptest.SetupControllerCase("/riders/application", http.MethodGet, nil)
	tc = withRiderSession(tc, userID)

	svc.On("GetOwnApplication", tc.Req.Context(), userID).Return(dtos.RiderApplicationResponse{
		Status:  "draft",
		CanEdit: true,
	}, nil)

	err := h.GetApplication(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(tc.Res.Body.Bytes(), &body))
	assert.Equal(t, "draft", body["status"])
	assert.Equal(t, true, body["can_edit"])
	assert.NotContains(t, body, "notes")
}

func TestGetApplication_AlreadyRider(t *testing.T) {
	svc := mocksriders.NewRiderProfileApp(t)
	h := ridershandler.NewRiderApplicationHandler(svc)
	userID := uuid.New()

	tc := buildhttptest.SetupControllerCase("/riders/application", http.MethodGet, nil)
	tc = withRiderSession(tc, userID)

	svc.On("GetOwnApplication", tc.Req.Context(), userID).Return(dtos.RiderApplicationResponse{}, errcustom.ErrAlreadyRider)

	err := h.GetApplication(tc.Ctx)

	assert.Equal(t, http.StatusConflict, httpCode(err))
}

func TestGetApplication_UnexpectedError(t *testing.T) {
	svc := mocksriders.NewRiderProfileApp(t)
	h := ridershandler.NewRiderApplicationHandler(svc)
	userID := uuid.New()

	tc := buildhttptest.SetupControllerCase("/riders/application", http.MethodGet, nil)
	tc = withRiderSession(tc, userID)

	svc.On("GetOwnApplication", tc.Req.Context(), userID).Return(dtos.RiderApplicationResponse{}, errcustom.ErrUnexpectedError)

	err := h.GetApplication(tc.Ctx)

	assert.Equal(t, http.StatusInternalServerError, httpCode(err))
}

func TestSaveApplication_Success(t *testing.T) {
	svc := mocksriders.NewRiderProfileApp(t)
	h := ridershandler.NewRiderApplicationHandler(svc)
	userID := uuid.New()

	tc := buildhttptest.SetupControllerCase("/riders/application", http.MethodPut, buildBody(t, map[string]any{
		"vehicle_type":  "moto",
		"vehicle_plate": "AB123CD",
		"notes":         "intento de escribir notas internas",
	}))
	tc = withRiderSession(tc, userID)

	svc.On("SaveOwnApplication", tc.Req.Context(), userID, mock.MatchedBy(func(req dtos.RiderApplicationRequest) bool {
		return *req.VehicleType == "moto" && *req.VehiclePlate == "AB123CD"
	})).Return(dtos.RiderApplicationResponse{Status: "draft", CanEdit: true}, nil)

	err := h.SaveApplication(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
}

func TestSaveApplication_InvalidBody(t *testing.T) {
	svc := mocksriders.NewRiderProfileApp(t)
	h := ridershandler.NewRiderApplicationHandler(svc)

	tc := buildhttptest.SetupControllerCase("/riders/application", http.MethodPut, bytes.NewBufferString("{not json"))
	tc = withRiderSession(tc, uuid.New())

	err := h.SaveApplication(tc.Ctx)

	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestSaveApplication_InvalidFieldsAreRejected(t *testing.T) {
	cases := map[string]map[string]any{
		"vehicle_type fuera del catálogo": {"vehicle_type": "camion"},
		"placa demasiado larga":           {"vehicle_plate": strings.Repeat("A", 16)},
		"foto demasiado grande":           {"id_photo": strings.Repeat("A", 7000001)},
	}
	for name, payload := range cases {
		t.Run(name, func(t *testing.T) {
			svc := mocksriders.NewRiderProfileApp(t)
			h := ridershandler.NewRiderApplicationHandler(svc)

			tc := buildhttptest.SetupControllerCase("/riders/application", http.MethodPut, buildBody(t, payload))
			tc = withRiderSession(tc, uuid.New())

			err := h.SaveApplication(tc.Ctx)

			assert.Equal(t, http.StatusBadRequest, httpCode(err))
		})
	}
}

func TestSaveApplication_Locked(t *testing.T) {
	svc := mocksriders.NewRiderProfileApp(t)
	h := ridershandler.NewRiderApplicationHandler(svc)
	userID := uuid.New()

	tc := buildhttptest.SetupControllerCase("/riders/application", http.MethodPut, buildBody(t, map[string]any{"vehicle_type": "moto"}))
	tc = withRiderSession(tc, userID)

	svc.On("SaveOwnApplication", tc.Req.Context(), userID, mock.AnythingOfType("RiderApplicationRequest")).
		Return(dtos.RiderApplicationResponse{}, errcustom.ErrRiderApplicationLocked)

	err := h.SaveApplication(tc.Ctx)

	assert.Equal(t, http.StatusConflict, httpCode(err))
}

func TestSaveApplication_UploadFails(t *testing.T) {
	svc := mocksriders.NewRiderProfileApp(t)
	h := ridershandler.NewRiderApplicationHandler(svc)
	userID := uuid.New()

	tc := buildhttptest.SetupControllerCase("/riders/application", http.MethodPut, buildBody(t, map[string]any{"id_photo": "data:image/png;base64,AAA"}))
	tc = withRiderSession(tc, userID)

	svc.On("SaveOwnApplication", tc.Req.Context(), userID, mock.AnythingOfType("RiderApplicationRequest")).
		Return(dtos.RiderApplicationResponse{}, errcustom.ErrUploadingImage)

	err := h.SaveApplication(tc.Ctx)

	assert.Equal(t, http.StatusUnprocessableEntity, httpCode(err))
}

func TestSubmitApplication_Success(t *testing.T) {
	svc := mocksriders.NewRiderProfileApp(t)
	h := ridershandler.NewRiderApplicationHandler(svc)
	userID := uuid.New()

	tc := buildhttptest.SetupControllerCase("/riders/application/submit", http.MethodPost, nil)
	tc = withRiderSession(tc, userID)

	svc.On("SubmitOwnApplication", tc.Req.Context(), userID).
		Return(dtos.RiderApplicationResponse{Status: "pending_review"}, nil)

	err := h.SubmitApplication(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
}

func TestSubmitApplication_IncompleteReturnsMissing(t *testing.T) {
	svc := mocksriders.NewRiderProfileApp(t)
	h := ridershandler.NewRiderApplicationHandler(svc)
	userID := uuid.New()

	tc := buildhttptest.SetupControllerCase("/riders/application/submit", http.MethodPost, nil)
	tc = withRiderSession(tc, userID)

	svc.On("SubmitOwnApplication", tc.Req.Context(), userID).Return(dtos.RiderApplicationResponse{
		Status:     "draft",
		Completion: dtos.RiderCompletionDTO{Missing: []string{"id_photo", "vehicle_plate"}},
	}, errcustom.ErrRiderProfileIncomplete)

	err := h.SubmitApplication(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, tc.Res.Code)
	var body dtos.ActivateErrorResponse
	require.NoError(t, json.Unmarshal(tc.Res.Body.Bytes(), &body))
	assert.Equal(t, []string{"id_photo", "vehicle_plate"}, body.Missing)
}

func TestSubmitApplication_Locked(t *testing.T) {
	svc := mocksriders.NewRiderProfileApp(t)
	h := ridershandler.NewRiderApplicationHandler(svc)
	userID := uuid.New()

	tc := buildhttptest.SetupControllerCase("/riders/application/submit", http.MethodPost, nil)
	tc = withRiderSession(tc, userID)

	svc.On("SubmitOwnApplication", tc.Req.Context(), userID).
		Return(dtos.RiderApplicationResponse{}, errcustom.ErrRiderApplicationLocked)

	err := h.SubmitApplication(tc.Ctx)

	assert.Equal(t, http.StatusConflict, httpCode(err))
}

func TestSubmitApplication_AlreadyRider(t *testing.T) {
	svc := mocksriders.NewRiderProfileApp(t)
	h := ridershandler.NewRiderApplicationHandler(svc)
	userID := uuid.New()

	tc := buildhttptest.SetupControllerCase("/riders/application/submit", http.MethodPost, nil)
	tc = withRiderSession(tc, userID)

	svc.On("SubmitOwnApplication", tc.Req.Context(), userID).
		Return(dtos.RiderApplicationResponse{}, errcustom.ErrAlreadyRider)

	err := h.SubmitApplication(tc.Ctx)

	assert.Equal(t, http.StatusConflict, httpCode(err))
}
