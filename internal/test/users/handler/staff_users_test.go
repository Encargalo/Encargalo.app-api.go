package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"

	authModels "Encargalo.app-api.go/internal/auth/domain/models"
	mocksauthports "Encargalo.app-api.go/internal/mocks/auth/domain/ports"
	mocksusersports "Encargalo.app-api.go/internal/mocks/users/domain/ports"
	"Encargalo.app-api.go/internal/shared/buildhttptest"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/users/domain/dto"
	usershandler "Encargalo.app-api.go/internal/users/handler"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func newStaffUsersHandler(t *testing.T) (usershandler.StaffUsersHandler, *mocksusersports.StaffUsersApp, *mocksauthports.AuthApp) {
	t.Helper()
	svc := mocksusersports.NewStaffUsersApp(t)
	auth := mocksauthports.NewAuthApp(t)
	return usershandler.NewStaffUsersHandler(svc, auth), svc, auth
}

func withStaffSession(tc buildhttptest.ControllerCase, session authModels.ActiveSession) buildhttptest.ControllerCase {
	ctx := context.WithValue(tc.Req.Context(), authModels.SessionKey, session)
	tc.Req = tc.Req.WithContext(ctx)
	tc.Ctx.SetRequest(tc.Req)
	return tc
}

func staffHTTPCode(err error) int {
	if he, ok := err.(*echo.HTTPError); ok {
		return he.Code
	}
	return 0
}

func TestStaffUsers_FindCandidate_MissingPhone(t *testing.T) {
	h, _, _ := newStaffUsersHandler(t)
	tc := buildhttptest.SetupControllerCase("/users/staff/candidates", http.MethodGet, nil)
	tc = withStaffSession(tc, authModels.ActiveSession{UserID: uuid.New()})

	err := h.FindCandidate(tc.Ctx)

	assert.Equal(t, http.StatusBadRequest, staffHTTPCode(err))
}

func TestStaffUsers_FindCandidate_Success(t *testing.T) {
	h, svc, _ := newStaffUsersHandler(t)
	userID := uuid.New()
	tc := buildhttptest.SetupControllerCase("/users/staff/candidates?phone=%2B573001112233", http.MethodGet, nil)
	tc = withStaffSession(tc, authModels.ActiveSession{UserID: uuid.New()})

	svc.On("FindCandidate", tc.Req.Context(), "+573001112233").Return(dto.StaffUserCandidateResponse{
		UserID: userID,
		Name:   "Ana",
		Phone:  "+573001112233",
	}, nil)

	err := h.FindCandidate(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
}

func TestStaffUsers_UpdateUser_NoFields(t *testing.T) {
	h, _, _ := newStaffUsersHandler(t)
	userID := uuid.New()
	tc := buildhttptest.SetupControllerCase("/users/staff/"+userID.String(), http.MethodPatch, bytes.NewBufferString("{}"))
	tc = withStaffSession(tc, authModels.ActiveSession{UserID: uuid.New()})
	tc.Ctx.SetParamNames("userId")
	tc.Ctx.SetParamValues(userID.String())

	err := h.UpdateUser(tc.Ctx)

	assert.Equal(t, http.StatusBadRequest, staffHTTPCode(err))
}

func TestStaffUsers_UpdateUser_Success(t *testing.T) {
	h, svc, _ := newStaffUsersHandler(t)
	staffID := uuid.New()
	userID := uuid.New()
	name := "Nuevo"
	body, _ := json.Marshal(dto.StaffUpdateUserRequest{Name: &name})
	tc := buildhttptest.SetupControllerCase("/users/staff/"+userID.String(), http.MethodPatch, bytes.NewBuffer(body))
	tc = withStaffSession(tc, authModels.ActiveSession{UserID: staffID})
	tc.Ctx.SetParamNames("userId")
	tc.Ctx.SetParamValues(userID.String())

	svc.On("UpdateUser", tc.Req.Context(), staffID, userID, dto.StaffUpdateUserRequest{Name: &name}).
		Return(dto.StaffUserDetailResponse{ID: userID, Name: name}, nil)

	err := h.UpdateUser(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
}

func TestStaffUsers_BlockUser_InvalidatesSessions(t *testing.T) {
	h, svc, auth := newStaffUsersHandler(t)
	staffID := uuid.New()
	userID := uuid.New()
	tc := buildhttptest.SetupControllerCase("/users/staff/"+userID.String()+"/block", http.MethodPost, nil)
	tc = withStaffSession(tc, authModels.ActiveSession{UserID: staffID})
	tc.Ctx.SetParamNames("userId")
	tc.Ctx.SetParamValues(userID.String())

	svc.On("BlockUser", tc.Req.Context(), staffID, userID).
		Return(dto.StaffUserDetailResponse{ID: userID, Enabled: false}, nil)
	auth.On("DeleteAllSessionsForUser", tc.Req.Context(), userID).Return(nil)

	err := h.BlockUser(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
}

func TestStaffUsers_BlockUser_NotFound(t *testing.T) {
	h, svc, _ := newStaffUsersHandler(t)
	userID := uuid.New()
	tc := buildhttptest.SetupControllerCase("/users/staff/"+userID.String()+"/block", http.MethodPost, nil)
	tc = withStaffSession(tc, authModels.ActiveSession{UserID: uuid.New()})
	tc.Ctx.SetParamNames("userId")
	tc.Ctx.SetParamValues(userID.String())

	svc.On("BlockUser", tc.Req.Context(), tc.Req.Context().Value(authModels.SessionKey).(authModels.ActiveSession).UserID, userID).
		Return(dto.StaffUserDetailResponse{}, errcustom.ErrNotFound)

	err := h.BlockUser(tc.Ctx)

	assert.Equal(t, http.StatusNotFound, staffHTTPCode(err))
}

func TestStaffUsers_ListUsers_Success(t *testing.T) {
	h, svc, _ := newStaffUsersHandler(t)
	tc := buildhttptest.SetupControllerCase("/users/staff?limit=10", http.MethodGet, nil)

	svc.On("ListUsers", tc.Req.Context(), dto.ListStaffUsersQuery{Limit: 10}).Return(dto.StaffUserListResponse{
		Items:   []dto.StaffUserListItem{},
		HasMore: false,
	}, nil)

	err := h.ListUsers(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
}
