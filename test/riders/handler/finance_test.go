package handler_test

import (
	"context"
	"net/http"
	"testing"

	authmodels "Encargalo.app-api.go/internal/auth/domain/models"
	mockriders "Encargalo.app-api.go/internal/mocks/riders/domain/ports"
	"Encargalo.app-api.go/internal/riders/domain/dtos"
	"Encargalo.app-api.go/internal/riders/domain/models"
	ridershandler "Encargalo.app-api.go/internal/riders/handler"
	"Encargalo.app-api.go/internal/shared/buildhttptest"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newFinanceHandler(t *testing.T) (ridershandler.FinanceHandler, *mockriders.FinanceApp) {
	t.Helper()
	svc := mockriders.NewFinanceApp(t)
	return ridershandler.NewFinanceHandler(svc), svc
}

func withRiderSession(tc buildhttptest.ControllerCase, riderID uuid.UUID) buildhttptest.ControllerCase {
	session := authmodels.ActiveSession{UserID: riderID}
	ctx := context.WithValue(tc.Req.Context(), authmodels.SessionKey, session)
	tc.Req = tc.Req.WithContext(ctx)
	tc.Ctx.SetRequest(tc.Req)
	return tc
}

// --- GetBalance ---

func TestGetBalance_Success(t *testing.T) {
	h, svc := newFinanceHandler(t)
	riderID := uuid.New()

	resp := dtos.BalanceResponse{
		Balance:      25.0,
		Zone:         models.ZoneWithdrawalAvailable,
		Transactions: []dtos.TransactionDTO{},
	}
	svc.On("GetBalance", mock.Anything, riderID).Return(resp, nil)

	tc := buildhttptest.SetupControllerCase("/riders/balance", http.MethodGet, nil)
	tc = withRiderSession(tc, riderID)

	err := h.GetBalance(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
}

func TestGetBalance_ServiceError(t *testing.T) {
	h, svc := newFinanceHandler(t)
	riderID := uuid.New()

	svc.On("GetBalance", mock.Anything, riderID).Return(dtos.BalanceResponse{}, errcustom.ErrUnexpectedError)

	tc := buildhttptest.SetupControllerCase("/riders/balance", http.MethodGet, nil)
	tc = withRiderSession(tc, riderID)

	err := h.GetBalance(tc.Ctx)

	assert.Error(t, err)
	he, ok := err.(*echo.HTTPError)
	if assert.True(t, ok) {
		assert.Equal(t, http.StatusInternalServerError, he.Code)
	}
}

// --- RequestWithdrawal ---

func TestRequestWithdrawal_Success(t *testing.T) {
	h, svc := newFinanceHandler(t)
	riderID := uuid.New()

	resp := dtos.WithdrawalResponse{AmountWithdrawn: 20.0}
	svc.On("RequestWithdrawal", mock.Anything, riderID).Return(resp, nil)

	tc := buildhttptest.SetupControllerCase("/riders/withdrawal", http.MethodPost, nil)
	tc = withRiderSession(tc, riderID)

	err := h.RequestWithdrawal(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
}

func TestRequestWithdrawal_InsufficientBalance(t *testing.T) {
	h, svc := newFinanceHandler(t)
	riderID := uuid.New()

	svc.On("RequestWithdrawal", mock.Anything, riderID).Return(dtos.WithdrawalResponse{}, errcustom.ErrInsufficientBalance)

	tc := buildhttptest.SetupControllerCase("/riders/withdrawal", http.MethodPost, nil)
	tc = withRiderSession(tc, riderID)

	err := h.RequestWithdrawal(tc.Ctx)

	assert.Error(t, err)
	he, ok := err.(*echo.HTTPError)
	if assert.True(t, ok) {
		assert.Equal(t, http.StatusUnprocessableEntity, he.Code)
	}
}

// --- ListTransactions ---

func TestListTransactions_Success(t *testing.T) {
	h, svc := newFinanceHandler(t)
	riderID := uuid.New()

	resp := dtos.TransactionsResponse{
		Transactions: []dtos.TransactionDTO{},
		Total:        0,
		Page:         1,
		Limit:        20,
	}
	svc.On("ListTransactions", mock.Anything, riderID, 1, 20).Return(resp, nil)

	tc := buildhttptest.SetupControllerCase("/riders/transactions", http.MethodGet, nil)
	tc = withRiderSession(tc, riderID)

	err := h.ListTransactions(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
}

func TestListTransactions_CustomPagination(t *testing.T) {
	h, svc := newFinanceHandler(t)
	riderID := uuid.New()

	resp := dtos.TransactionsResponse{
		Transactions: []dtos.TransactionDTO{},
		Total:        0,
		Page:         2,
		Limit:        10,
	}
	svc.On("ListTransactions", mock.Anything, riderID, 2, 10).Return(resp, nil)

	tc := buildhttptest.SetupControllerCase("/riders/transactions?page=2&limit=10", http.MethodGet, nil)
	tc.Req.URL.RawQuery = "page=2&limit=10"
	tc.Ctx.SetRequest(tc.Req)
	tc = withRiderSession(tc, riderID)

	err := h.ListTransactions(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
}
