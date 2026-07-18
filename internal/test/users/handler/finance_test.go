package handler_test

import (
	"context"
	"net/http"
	"testing"

	authmodels "Encargalo.app-api.go/internal/auth/domain/models"
	mocksusers "Encargalo.app-api.go/internal/mocks/users/domain/ports"
	"Encargalo.app-api.go/internal/shared/buildhttptest"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/users/domain/dto"
	usershandler "Encargalo.app-api.go/internal/users/handler"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func withSessionFinance(tc buildhttptest.ControllerCase, session authmodels.ActiveSession) buildhttptest.ControllerCase {
	ctx := context.WithValue(tc.Req.Context(), authmodels.SessionKey, session)
	tc.Req = tc.Req.WithContext(ctx)
	tc.Ctx.SetRequest(tc.Req)
	return tc
}

func TestGetBalance_Success(t *testing.T) {
	svc := mocksusers.NewCustomerFinanceApp(t)
	h := usershandler.NewCustomerFinanceHandler(svc)

	userID := uuid.New()
	tc := buildhttptest.SetupControllerCase("/users/balance", http.MethodGet, nil)
	tc = withSessionFinance(tc, authmodels.ActiveSession{UserID: userID})

	svc.On("GetBalance", mock.Anything, userID).Return(dto.CustomerBalanceResponse{
		Balance:      12.5,
		Transactions: nil,
	}, nil)

	err := h.GetBalance(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
}

func TestGetBalance_NoSession(t *testing.T) {
	svc := mocksusers.NewCustomerFinanceApp(t)
	h := usershandler.NewCustomerFinanceHandler(svc)

	tc := buildhttptest.SetupControllerCase("/users/balance", http.MethodGet, nil)

	err := h.GetBalance(tc.Ctx)

	assert.Error(t, err)
	assert.ErrorIs(t, err, errcustom.ErrUnautorized)
}

func TestGetBalance_ServiceError(t *testing.T) {
	svc := mocksusers.NewCustomerFinanceApp(t)
	h := usershandler.NewCustomerFinanceHandler(svc)

	userID := uuid.New()
	tc := buildhttptest.SetupControllerCase("/users/balance", http.MethodGet, nil)
	tc = withSessionFinance(tc, authmodels.ActiveSession{UserID: userID})

	svc.On("GetBalance", mock.Anything, userID).Return(dto.CustomerBalanceResponse{}, errcustom.ErrUnexpectedError)

	err := h.GetBalance(tc.Ctx)

	assert.Error(t, err)
}
