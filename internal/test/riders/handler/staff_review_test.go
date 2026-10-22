package handler_test

import (
	"context"
	"net/http"
	"testing"

	authModels "Encargalo.app-api.go/internal/auth/domain/models"
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

// reviewCase arma la request de decisión con el staff autenticado en sesión.
func reviewCase(t *testing.T, path string, userID uuid.UUID, staffID uuid.UUID, body any) buildhttptest.ControllerCase {
	t.Helper()
	tc := buildhttptest.SetupControllerCase(path, http.MethodPost, buildBody(t, body))
	tc.Req = tc.Req.WithContext(
		context.WithValue(tc.Req.Context(), authModels.SessionKey, authModels.ActiveSession{UserID: staffID}),
	)
	tc.Ctx.SetRequest(tc.Req)
	tc.Ctx.SetParamNames("userId")
	tc.Ctx.SetParamValues(userID.String())
	return tc
}

func TestRequestChanges_Success(t *testing.T) {
	svc := mocksriders.NewRiderProfileApp(t)
	h := ridershandler.NewStaffRiderHandler(svc)
	userID, staffID := uuid.New(), uuid.New()
	reason := "La foto de la licencia está borrosa"

	tc := reviewCase(t, "/riders/staff/profiles/"+userID.String()+"/request-changes", userID, staffID,
		dtos.ReviewApplicationRequest{Reason: reason})

	svc.On("RequestApplicationChanges", mock.Anything, userID, staffID, reason).
		Return(dtos.RiderProfileResponse{UserID: userID.String(), Status: "changes_requested"}, nil)

	err := h.RequestChanges(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
}

func TestReject_Success(t *testing.T) {
	svc := mocksriders.NewRiderProfileApp(t)
	h := ridershandler.NewStaffRiderHandler(svc)
	userID, staffID := uuid.New(), uuid.New()
	reason := "Los documentos no corresponden al titular"

	tc := reviewCase(t, "/riders/staff/profiles/"+userID.String()+"/reject", userID, staffID,
		dtos.ReviewApplicationRequest{Reason: reason})

	svc.On("RejectApplication", mock.Anything, userID, staffID, reason).
		Return(dtos.RiderProfileResponse{UserID: userID.String(), Status: "rejected"}, nil)

	err := h.Reject(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
}

func TestReview_ReasonIsRequired(t *testing.T) {
	for name, reason := range map[string]string{"vacío": "", "muy corto": "no"} {
		t.Run(name, func(t *testing.T) {
			svc := mocksriders.NewRiderProfileApp(t)
			h := ridershandler.NewStaffRiderHandler(svc)
			userID, staffID := uuid.New(), uuid.New()

			tc := reviewCase(t, "/riders/staff/profiles/"+userID.String()+"/request-changes", userID, staffID,
				dtos.ReviewApplicationRequest{Reason: reason})

			err := h.RequestChanges(tc.Ctx)

			require.Error(t, err)
			assert.Equal(t, http.StatusBadRequest, httpCode(err))
		})
	}
}

func TestReview_InvalidUserID(t *testing.T) {
	svc := mocksriders.NewRiderProfileApp(t)
	h := ridershandler.NewStaffRiderHandler(svc)

	tc := reviewCase(t, "/riders/staff/profiles/not-a-uuid/reject", uuid.New(), uuid.New(),
		dtos.ReviewApplicationRequest{Reason: "Un motivo cualquiera"})
	tc.Ctx.SetParamValues("not-a-uuid")

	err := h.Reject(tc.Ctx)

	require.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestReview_NotPendingIsConflict(t *testing.T) {
	svc := mocksriders.NewRiderProfileApp(t)
	h := ridershandler.NewStaffRiderHandler(svc)
	userID, staffID := uuid.New(), uuid.New()
	reason := "Un motivo cualquiera"

	tc := reviewCase(t, "/riders/staff/profiles/"+userID.String()+"/reject", userID, staffID,
		dtos.ReviewApplicationRequest{Reason: reason})

	svc.On("RejectApplication", mock.Anything, userID, staffID, reason).
		Return(dtos.RiderProfileResponse{}, errcustom.ErrRiderReviewNotPending)

	err := h.Reject(tc.Ctx)

	require.Error(t, err)
	assert.Equal(t, http.StatusConflict, httpCode(err))
}

func TestReview_MissingProfileIsNotFound(t *testing.T) {
	svc := mocksriders.NewRiderProfileApp(t)
	h := ridershandler.NewStaffRiderHandler(svc)
	userID, staffID := uuid.New(), uuid.New()
	reason := "Un motivo cualquiera"

	tc := reviewCase(t, "/riders/staff/profiles/"+userID.String()+"/request-changes", userID, staffID,
		dtos.ReviewApplicationRequest{Reason: reason})

	svc.On("RequestApplicationChanges", mock.Anything, userID, staffID, reason).
		Return(dtos.RiderProfileResponse{}, errcustom.ErrNotFound)

	err := h.RequestChanges(tc.Ctx)

	require.Error(t, err)
	assert.Equal(t, http.StatusNotFound, httpCode(err))
}
