package handler_test

import (
	"bytes"
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
	"github.com/labstack/echo/v4"
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
		BalanceBs:    12.5,
		BalanceUSD:   0.32,
		Transactions: nil,
	}, nil)

	err := h.GetBalance(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
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

func TestUpsertPagoMovil_Success(t *testing.T) {
	svc := mocksusers.NewCustomerFinanceApp(t)
	h := usershandler.NewCustomerFinanceHandler(svc)

	userID := uuid.New()
	body := bytes.NewReader([]byte(`{"phone":"+584121234567","bank_code":"0102","national_id":"V12345678"}`))
	tc := buildhttptest.SetupControllerCase("/users/pago-movil", http.MethodPut, body)
	tc = withSessionFinance(tc, authmodels.ActiveSession{UserID: userID})

	svc.On("UpsertPagoMovil", mock.Anything, userID, mock.AnythingOfType("dto.UpsertPagoMovilRequest")).
		Return(dto.PagoMovilResponse{Phone: "+584121234567", BankCode: "0102", NationalID: "V12345678"}, nil)

	err := h.UpsertPagoMovil(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
}

func TestCreateRefundRequest_Success(t *testing.T) {
	svc := mocksusers.NewCustomerFinanceApp(t)
	h := usershandler.NewCustomerFinanceHandler(svc)

	userID := uuid.New()
	txID := uuid.New()
	body := bytes.NewReader([]byte(`{"transaction_id":"` + txID.String() + `"}`))
	tc := buildhttptest.SetupControllerCase("/users/balance/refund-requests", http.MethodPost, body)
	tc = withSessionFinance(tc, authmodels.ActiveSession{UserID: userID})

	svc.On("CreateRefundRequest", mock.Anything, userID, txID).Return(dto.RefundRequestResponse{
		ID:     uuid.New().String(),
		Status: "pending",
	}, nil)

	err := h.CreateRefundRequest(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, tc.Res.Code)
}

func TestCreateRefundRequest_NoPagoMovil(t *testing.T) {
	svc := mocksusers.NewCustomerFinanceApp(t)
	h := usershandler.NewCustomerFinanceHandler(svc)

	userID := uuid.New()
	txID := uuid.New()
	body := bytes.NewReader([]byte(`{"transaction_id":"` + txID.String() + `"}`))
	tc := buildhttptest.SetupControllerCase("/users/balance/refund-requests", http.MethodPost, body)
	tc = withSessionFinance(tc, authmodels.ActiveSession{UserID: userID})

	svc.On("CreateRefundRequest", mock.Anything, userID, txID).Return(dto.RefundRequestResponse{}, errcustom.ErrPagoMovilRequired)

	err := h.CreateRefundRequest(tc.Ctx)

	assert.Error(t, err)
	he, ok := err.(*echo.HTTPError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusUnprocessableEntity, he.Code)
}

func TestGetRefundRequest_Success(t *testing.T) {
	svc := mocksusers.NewCustomerFinanceApp(t)
	h := usershandler.NewCustomerFinanceHandler(svc)

	userID := uuid.New()
	reqID := uuid.New()
	tc := buildhttptest.SetupControllerCase("/users/balance/refund-requests/"+reqID.String(), http.MethodGet, nil)
	tc = withSessionFinance(tc, authmodels.ActiveSession{UserID: userID})
	tc.Ctx.SetParamNames("id")
	tc.Ctx.SetParamValues(reqID.String())

	proof := "https://cdn.example/proof.jpg"
	svc.On("GetRefundRequest", mock.Anything, userID, reqID).Return(dto.RefundRequestResponse{
		ID:       reqID.String(),
		Status:   "completed",
		ProofURL: &proof,
	}, nil)

	err := h.GetRefundRequest(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
}
