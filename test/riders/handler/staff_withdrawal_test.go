package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	authmodels "Encargalo.app-api.go/internal/auth/domain/models"
	financedtos "Encargalo.app-api.go/internal/finance/domain/dtos"
	financeModels "Encargalo.app-api.go/internal/finance/domain/models"
	mockfinance "Encargalo.app-api.go/internal/mocks/finance/domain/ports"
	ridershandler "Encargalo.app-api.go/internal/riders/handler"
	"Encargalo.app-api.go/internal/shared/buildhttptest"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func jsonBody(t *testing.T, payload map[string]string) io.Reader {
	t.Helper()
	raw, err := json.Marshal(payload)
	require.NoError(t, err)
	return bytes.NewReader(raw)
}

func newStaffWithdrawalHandler(t *testing.T) (ridershandler.StaffWithdrawalHandler, *mockfinance.WithdrawalApp) {
	t.Helper()
	svc := mockfinance.NewWithdrawalApp(t)
	return ridershandler.NewStaffWithdrawalHandler(svc), svc
}

func withStaffSession(tc buildhttptest.ControllerCase, staffID uuid.UUID) buildhttptest.ControllerCase {
	session := authmodels.ActiveSession{UserID: staffID}
	ctx := context.WithValue(tc.Req.Context(), authmodels.SessionKey, session)
	tc.Req = tc.Req.WithContext(ctx)
	tc.Ctx.SetRequest(tc.Req)
	return tc
}

// Sin filtro el listado trae las activas: son las que hay que atender.
func TestListWithdrawals_DefaultsToActiveStatuses(t *testing.T) {
	h, svc := newStaffWithdrawalHandler(t)

	svc.On("List", mock.Anything, financeModels.EntityTypeRider,
		[]financeModels.WithdrawalStatus{financeModels.WithdrawalPending, financeModels.WithdrawalProcessing}).
		Return(financedtos.WithdrawalsResponse{Total: 0}, nil)

	tc := buildhttptest.SetupControllerCase("/riders/staff/finance/withdrawals", http.MethodGet, nil)

	err := h.List(tc.Ctx)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
}

func TestListWithdrawals_FiltersByStatus(t *testing.T) {
	h, svc := newStaffWithdrawalHandler(t)

	svc.On("List", mock.Anything, financeModels.EntityTypeRider,
		[]financeModels.WithdrawalStatus{financeModels.WithdrawalCompleted}).
		Return(financedtos.WithdrawalsResponse{Total: 0}, nil)

	tc := buildhttptest.SetupControllerCase("/riders/staff/finance/withdrawals", http.MethodGet, nil)
	tc.Ctx.QueryParams().Set("status", "completed")

	err := h.List(tc.Ctx)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
}

func TestListWithdrawals_InvalidStatus(t *testing.T) {
	h, _ := newStaffWithdrawalHandler(t)

	tc := buildhttptest.SetupControllerCase("/riders/staff/finance/withdrawals", http.MethodGet, nil)
	tc.Ctx.QueryParams().Set("status", "pagado")

	err := h.List(tc.Ctx)

	he, ok := err.(*echo.HTTPError)
	if assert.True(t, ok) {
		assert.Equal(t, http.StatusBadRequest, he.Code)
	}
}

func TestClaimWithdrawal_ReturnsAmountToTransfer(t *testing.T) {
	h, svc := newStaffWithdrawalHandler(t)
	requestID := uuid.New()
	staffID := uuid.New()

	svc.On("Claim", mock.Anything, requestID, staffID).
		Return(financedtos.ClaimResponse{ID: requestID.String(), PayableBs: 72.0, Status: "processing"}, nil)

	tc := buildhttptest.SetupControllerCase("/riders/staff/finance/withdrawals/:id/claim", http.MethodPost, nil)
	tc.Ctx.SetParamNames("id")
	tc.Ctx.SetParamValues(requestID.String())
	tc = withStaffSession(tc, staffID)

	err := h.Claim(tc.Ctx)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
}

// Reclamar dos veces es el caso de dos operadores sobre la misma solicitud.
func TestClaimWithdrawal_AlreadyClaimed(t *testing.T) {
	h, svc := newStaffWithdrawalHandler(t)
	requestID := uuid.New()
	staffID := uuid.New()

	svc.On("Claim", mock.Anything, requestID, staffID).
		Return(financedtos.ClaimResponse{}, errcustom.ErrWithdrawalInvalidTransition)

	tc := buildhttptest.SetupControllerCase("/riders/staff/finance/withdrawals/:id/claim", http.MethodPost, nil)
	tc.Ctx.SetParamNames("id")
	tc.Ctx.SetParamValues(requestID.String())
	tc = withStaffSession(tc, staffID)

	err := h.Claim(tc.Ctx)

	he, ok := err.(*echo.HTTPError)
	if assert.True(t, ok) {
		assert.Equal(t, http.StatusConflict, he.Code)
	}
}

func TestClaimWithdrawal_InvalidID(t *testing.T) {
	h, _ := newStaffWithdrawalHandler(t)

	tc := buildhttptest.SetupControllerCase("/riders/staff/finance/withdrawals/:id/claim", http.MethodPost, nil)
	tc.Ctx.SetParamNames("id")
	tc.Ctx.SetParamValues("no-es-uuid")

	err := h.Claim(tc.Ctx)

	he, ok := err.(*echo.HTTPError)
	if assert.True(t, ok) {
		assert.Equal(t, http.StatusBadRequest, he.Code)
	}
}

func TestClaimWithdrawal_NotFound(t *testing.T) {
	h, svc := newStaffWithdrawalHandler(t)
	requestID := uuid.New()
	staffID := uuid.New()

	svc.On("Claim", mock.Anything, requestID, staffID).
		Return(financedtos.ClaimResponse{}, errcustom.ErrWithdrawalNotFound)

	tc := buildhttptest.SetupControllerCase("/riders/staff/finance/withdrawals/:id/claim", http.MethodPost, nil)
	tc.Ctx.SetParamNames("id")
	tc.Ctx.SetParamValues(requestID.String())
	tc = withStaffSession(tc, staffID)

	err := h.Claim(tc.Ctx)

	he, ok := err.(*echo.HTTPError)
	if assert.True(t, ok) {
		assert.Equal(t, http.StatusNotFound, he.Code)
	}
}

func TestCompleteWithdrawal_ProofRequired(t *testing.T) {
	h, svc := newStaffWithdrawalHandler(t)
	requestID := uuid.New()
	staffID := uuid.New()

	svc.On("Complete", mock.Anything, requestID, staffID, mock.Anything).
		Return(financedtos.WithdrawalItem{}, errcustom.ErrWithdrawalProofRequired)

	tc := buildhttptest.SetupControllerCase("/riders/staff/finance/withdrawals/:id/complete", http.MethodPost, jsonBody(t, map[string]string{}))
	tc.Ctx.SetParamNames("id")
	tc.Ctx.SetParamValues(requestID.String())
	tc = withStaffSession(tc, staffID)

	err := h.Complete(tc.Ctx)

	he, ok := err.(*echo.HTTPError)
	if assert.True(t, ok) {
		assert.Equal(t, http.StatusBadRequest, he.Code)
	}
}

func TestCompleteWithdrawal_Success(t *testing.T) {
	h, svc := newStaffWithdrawalHandler(t)
	requestID := uuid.New()
	staffID := uuid.New()

	svc.On("Complete", mock.Anything, requestID, staffID, mock.Anything).
		Return(financedtos.WithdrawalItem{ID: requestID.String(), Status: "completed", PayableBs: 72.0}, nil)

	tc := buildhttptest.SetupControllerCase("/riders/staff/finance/withdrawals/:id/complete", http.MethodPost,
		jsonBody(t, map[string]string{"proof_image": "data:image/png;base64,abc"}))
	tc.Ctx.SetParamNames("id")
	tc.Ctx.SetParamValues(requestID.String())
	tc = withStaffSession(tc, staffID)

	err := h.Complete(tc.Ctx)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
}

func TestRejectWithdrawal_ReasonRequired(t *testing.T) {
	h, svc := newStaffWithdrawalHandler(t)
	requestID := uuid.New()
	staffID := uuid.New()

	svc.On("Reject", mock.Anything, requestID, staffID, mock.Anything).
		Return(financedtos.WithdrawalItem{}, errcustom.ErrWithdrawalReasonRequired)

	tc := buildhttptest.SetupControllerCase("/riders/staff/finance/withdrawals/:id/reject", http.MethodPost, jsonBody(t, map[string]string{}))
	tc.Ctx.SetParamNames("id")
	tc.Ctx.SetParamValues(requestID.String())
	tc = withStaffSession(tc, staffID)

	err := h.Reject(tc.Ctx)

	he, ok := err.(*echo.HTTPError)
	if assert.True(t, ok) {
		assert.Equal(t, http.StatusBadRequest, he.Code)
	}
}

func TestRejectWithdrawal_Success(t *testing.T) {
	h, svc := newStaffWithdrawalHandler(t)
	requestID := uuid.New()
	staffID := uuid.New()

	svc.On("Reject", mock.Anything, requestID, staffID, mock.Anything).
		Return(financedtos.WithdrawalItem{ID: requestID.String(), Status: "rejected"}, nil)

	tc := buildhttptest.SetupControllerCase("/riders/staff/finance/withdrawals/:id/reject", http.MethodPost,
		jsonBody(t, map[string]string{"reason": "el Pago Móvil no corresponde al rider"}))
	tc.Ctx.SetParamNames("id")
	tc.Ctx.SetParamValues(requestID.String())
	tc = withStaffSession(tc, staffID)

	err := h.Reject(tc.Ctx)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
}
