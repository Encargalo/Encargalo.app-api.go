package app_test

import (
	"context"
	"testing"
	"time"

	bcvmodels "Encargalo.app-api.go/internal/bcv/domain/models"
	mockbcvports "Encargalo.app-api.go/internal/mocks/bcv/domain/ports"
	mockports "Encargalo.app-api.go/internal/mocks/users/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/users/app"
	"Encargalo.app-api.go/internal/users/domain/dto"
	"Encargalo.app-api.go/internal/users/domain/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGetBalance_Success(t *testing.T) {
	repo := mockports.NewCustomerFinanceRepo(t)
	hub := mockports.NewStaffRefundHub(t)
	bcv := mockbcvports.NewBcvApp(t)
	svc := app.NewCustomerFinanceApp(repo, hub, nil, nil, bcv)

	customerID := uuid.New()
	repo.On("GetBalance", mock.Anything, customerID, models.EntityTypeCustomer).
		Return(models.Balance{EntityID: customerID, Amount: 400, Currency: models.CurrencyVES}, nil)
	repo.On("ListTransactions", mock.Anything, customerID, models.EntityTypeCustomer, 1, 10).
		Return([]models.Transaction{}, 0, nil)
	bcv.On("GetRate", mock.Anything).Return(&bcvmodels.ExchangeRate{Rate: 40}, nil)

	resp, err := svc.GetBalance(context.Background(), customerID)

	require.NoError(t, err)
	assert.Equal(t, 400.0, resp.BalanceBs)
	assert.Equal(t, 40.0, resp.BcvRate)
	assert.Empty(t, resp.Transactions)
}

func TestGetBalance_PopulatesOrderIDForOrderReference(t *testing.T) {
	repo := mockports.NewCustomerFinanceRepo(t)
	hub := mockports.NewStaffRefundHub(t)
	bcv := mockbcvports.NewBcvApp(t)
	svc := app.NewCustomerFinanceApp(repo, hub, nil, nil, bcv)

	customerID := uuid.New()
	orderID := uuid.New()
	refType := "order"
	tripID := uuid.New()
	tripType := "trip"
	repo.On("GetBalance", mock.Anything, customerID, models.EntityTypeCustomer).
		Return(models.Balance{EntityID: customerID, Amount: 100, Currency: models.CurrencyVES}, nil)
	repo.On("ListTransactions", mock.Anything, customerID, models.EntityTypeCustomer, 1, 10).
		Return([]models.Transaction{
			{
				ID: uuid.New(), ReferenceID: &orderID, ReferenceType: &refType,
				Amount: 10, Currency: models.CurrencyVES,
				MovementType: string(models.MovementTypePaymentOverage), CreatedAt: time.Now(),
			},
			{
				ID: uuid.New(), ReferenceID: &tripID, ReferenceType: &tripType,
				Amount: 5, Currency: models.CurrencyVES,
				MovementType: string(models.MovementTypePaymentOverage), CreatedAt: time.Now(),
			},
		}, 0, nil)
	bcv.On("GetRate", mock.Anything).Return(&bcvmodels.ExchangeRate{Rate: 40}, nil)

	resp, err := svc.GetBalance(context.Background(), customerID)

	require.NoError(t, err)
	require.Len(t, resp.Transactions, 2)
	require.NotNil(t, resp.Transactions[0].OrderID)
	assert.Equal(t, orderID.String(), *resp.Transactions[0].OrderID)
	assert.Nil(t, resp.Transactions[1].OrderID)
}

func TestGetBalance_RepoError(t *testing.T) {
	repo := mockports.NewCustomerFinanceRepo(t)
	hub := mockports.NewStaffRefundHub(t)
	svc := app.NewCustomerFinanceApp(repo, hub, nil, nil, nil)

	customerID := uuid.New()
	repo.On("GetBalance", mock.Anything, customerID, models.EntityTypeCustomer).
		Return(models.Balance{}, errcustom.ErrUnexpectedError)

	_, err := svc.GetBalance(context.Background(), customerID)

	assert.ErrorIs(t, err, errcustom.ErrUnexpectedError)
}

func TestGetBalanceBs_DelegatesToRepo(t *testing.T) {
	repo := mockports.NewCustomerFinanceRepo(t)
	hub := mockports.NewStaffRefundHub(t)
	svc := app.NewCustomerFinanceApp(repo, hub, nil, nil, nil)

	customerID := uuid.New()
	repo.On("GetBalance", mock.Anything, customerID, models.EntityTypeCustomer).
		Return(models.Balance{Amount: 123.45}, nil)

	got, err := svc.GetBalanceBs(context.Background(), customerID)

	require.NoError(t, err)
	assert.Equal(t, 123.45, got)
}

func TestGetBalancePaymentDebitForOrder_DelegatesToRepo(t *testing.T) {
	repo := mockports.NewCustomerFinanceRepo(t)
	hub := mockports.NewStaffRefundHub(t)
	svc := app.NewCustomerFinanceApp(repo, hub, nil, nil, nil)

	customerID := uuid.New()
	orderID := uuid.New()
	repo.On("SumBalancePaymentDebit", mock.Anything, customerID, orderID, "order").Return(50.0, nil)

	got, err := svc.GetBalancePaymentDebitForReference(context.Background(), customerID, orderID, "order")

	require.NoError(t, err)
	assert.Equal(t, 50.0, got)
}

func TestCreditPaymentOverage_RecordsRefundableMovement(t *testing.T) {
	repo := mockports.NewCustomerFinanceRepo(t)
	hub := mockports.NewStaffRefundHub(t)
	svc := app.NewCustomerFinanceApp(repo, hub, nil, nil, nil)

	customerID := uuid.New()
	orderID := uuid.New()
	repo.On("RecordMovement", mock.Anything, customerID, models.EntityTypeCustomer, 100.0, 0.0,
		mock.MatchedBy(func(tx models.Transaction) bool {
			return tx.MovementType == string(models.MovementTypePaymentOverage) &&
				tx.RefundablePMBs != nil && *tx.RefundablePMBs == 100.0 &&
				tx.ReferenceID != nil && *tx.ReferenceID == orderID &&
				tx.ReferenceType != nil && *tx.ReferenceType == "order"
		})).Return(0.0, nil)

	txID, createdAt, err := svc.CreditPaymentOverage(context.Background(), customerID, orderID, "order", 100.0, 40.0)

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, txID)
	assert.False(t, createdAt.IsZero())
}

func TestCreditPaymentOverage_ZeroAmountNoop(t *testing.T) {
	repo := mockports.NewCustomerFinanceRepo(t)
	hub := mockports.NewStaffRefundHub(t)
	svc := app.NewCustomerFinanceApp(repo, hub, nil, nil, nil)

	txID, createdAt, err := svc.CreditPaymentOverage(context.Background(), uuid.New(), uuid.New(), "order", 0, 40.0)

	require.NoError(t, err)
	assert.Equal(t, uuid.Nil, txID)
	assert.True(t, createdAt.IsZero())
}

func TestCreditPaymentPartial_RecordsRefundableMovement(t *testing.T) {
	repo := mockports.NewCustomerFinanceRepo(t)
	hub := mockports.NewStaffRefundHub(t)
	svc := app.NewCustomerFinanceApp(repo, hub, nil, nil, nil)

	customerID := uuid.New()
	orderID := uuid.New()
	repo.On("RecordMovement", mock.Anything, customerID, models.EntityTypeCustomer, 30.0, 0.0,
		mock.MatchedBy(func(tx models.Transaction) bool {
			return tx.MovementType == string(models.MovementTypePaymentPartialCredit)
		})).Return(0.0, nil)

	_, _, err := svc.CreditPaymentPartial(context.Background(), customerID, orderID, "order", 30.0, 40.0)

	require.NoError(t, err)
}

func TestCreditOrderRejected_UsesGivenRefundablePortion(t *testing.T) {
	repo := mockports.NewCustomerFinanceRepo(t)
	hub := mockports.NewStaffRefundHub(t)
	svc := app.NewCustomerFinanceApp(repo, hub, nil, nil, nil)

	customerID := uuid.New()
	orderID := uuid.New()
	repo.On("RecordMovement", mock.Anything, customerID, models.EntityTypeCustomer, 200.0, 0.0,
		mock.MatchedBy(func(tx models.Transaction) bool {
			return tx.MovementType == string(models.MovementTypeOrderRejectedRefund) &&
				tx.RefundablePMBs != nil && *tx.RefundablePMBs == 80.0
		})).Return(0.0, nil)

	_, _, err := svc.CreditOrderRejected(context.Background(), customerID, orderID, "order", 200.0, 80.0, 40.0)

	require.NoError(t, err)
}

func TestDebitForOrder_RecordsNegativeMovement(t *testing.T) {
	repo := mockports.NewCustomerFinanceRepo(t)
	hub := mockports.NewStaffRefundHub(t)
	svc := app.NewCustomerFinanceApp(repo, hub, nil, nil, nil)

	customerID := uuid.New()
	orderID := uuid.New()
	repo.On("RecordMovement", mock.Anything, customerID, models.EntityTypeCustomer, -60.0, 60.0,
		mock.MatchedBy(func(tx models.Transaction) bool {
			return tx.MovementType == string(models.MovementTypeBalancePayment)
		})).Return(0.0, nil)

	err := svc.DebitForReference(context.Background(), customerID, orderID, "order", 60.0)

	assert.NoError(t, err)
}

func TestDebitForOrder_ZeroAmountNoop(t *testing.T) {
	repo := mockports.NewCustomerFinanceRepo(t)
	hub := mockports.NewStaffRefundHub(t)
	svc := app.NewCustomerFinanceApp(repo, hub, nil, nil, nil)

	err := svc.DebitForReference(context.Background(), uuid.New(), uuid.New(), "order", 0)

	assert.NoError(t, err)
}

func TestRefundBalancePayment_RecordsPositiveMovement(t *testing.T) {
	repo := mockports.NewCustomerFinanceRepo(t)
	hub := mockports.NewStaffRefundHub(t)
	svc := app.NewCustomerFinanceApp(repo, hub, nil, nil, nil)

	customerID := uuid.New()
	orderID := uuid.New()
	repo.On("RecordMovement", mock.Anything, customerID, models.EntityTypeCustomer, 60.0, 0.0,
		mock.MatchedBy(func(tx models.Transaction) bool {
			return tx.MovementType == string(models.MovementTypeBalancePaymentRefund)
		})).Return(0.0, nil)

	err := svc.RefundBalancePayment(context.Background(), customerID, orderID, "order", 60.0)

	assert.NoError(t, err)
}

func TestGetPagoMovil_Success(t *testing.T) {
	repo := mockports.NewCustomerFinanceRepo(t)
	hub := mockports.NewStaffRefundHub(t)
	svc := app.NewCustomerFinanceApp(repo, hub, nil, nil, nil)

	customerID := uuid.New()
	repo.On("GetPagoMovil", mock.Anything, customerID).Return(&models.CustomerPagoMovil{
		Phone: "0414", BankCode: "0102", NationalID: "V1", UpdatedAt: time.Now(),
	}, nil)

	resp, err := svc.GetPagoMovil(context.Background(), customerID)

	require.NoError(t, err)
	assert.Equal(t, "0414", resp.Phone)
}

func TestGetPagoMovil_NotFound(t *testing.T) {
	repo := mockports.NewCustomerFinanceRepo(t)
	hub := mockports.NewStaffRefundHub(t)
	svc := app.NewCustomerFinanceApp(repo, hub, nil, nil, nil)

	customerID := uuid.New()
	repo.On("GetPagoMovil", mock.Anything, customerID).Return(nil, errcustom.ErrNotFound)

	_, err := svc.GetPagoMovil(context.Background(), customerID)

	assert.ErrorIs(t, err, errcustom.ErrNotFound)
}

func TestUpsertPagoMovil_Success(t *testing.T) {
	repo := mockports.NewCustomerFinanceRepo(t)
	hub := mockports.NewStaffRefundHub(t)
	svc := app.NewCustomerFinanceApp(repo, hub, nil, nil, nil)

	customerID := uuid.New()
	req := dto.UpsertPagoMovilRequest{Phone: "0414", BankCode: "0102", NationalID: "V1"}

	repo.On("UpsertPagoMovil", mock.Anything, mock.MatchedBy(func(pm *models.CustomerPagoMovil) bool {
		return pm.CustomerID == customerID && pm.Phone == "0414"
	})).Return(nil)
	repo.On("GetPagoMovil", mock.Anything, customerID).Return(&models.CustomerPagoMovil{
		Phone: "0414", BankCode: "0102", NationalID: "V1", UpdatedAt: time.Now(),
	}, nil)

	resp, err := svc.UpsertPagoMovil(context.Background(), customerID, req)

	require.NoError(t, err)
	assert.Equal(t, "0414", resp.Phone)
}

func TestUpsertPagoMovil_MissingFields(t *testing.T) {
	repo := mockports.NewCustomerFinanceRepo(t)
	hub := mockports.NewStaffRefundHub(t)
	svc := app.NewCustomerFinanceApp(repo, hub, nil, nil, nil)

	_, err := svc.UpsertPagoMovil(context.Background(), uuid.New(), dto.UpsertPagoMovilRequest{Phone: "", BankCode: "0102", NationalID: "V1"})

	assert.ErrorIs(t, err, errcustom.ErrInvalidBalanceAmount)
}

func TestListRefundRequests_Success(t *testing.T) {
	repo := mockports.NewCustomerFinanceRepo(t)
	hub := mockports.NewStaffRefundHub(t)
	svc := app.NewCustomerFinanceApp(repo, hub, nil, nil, nil)

	customerID := uuid.New()
	now := time.Now()
	repo.On("ListRefundRequestsByCustomer", mock.Anything, customerID).Return([]models.BalanceRefundRequest{
		{ID: uuid.New(), CustomerID: customerID, AmountBs: 50, Status: models.RefundPending, CreatedAt: now, ExpiresAt: now.Add(time.Hour)},
	}, nil)

	list, err := svc.ListRefundRequests(context.Background(), customerID)

	require.NoError(t, err)
	assert.Len(t, list, 1)
}

func TestGetRefundRequest_ForbiddenOtherCustomer(t *testing.T) {
	repo := mockports.NewCustomerFinanceRepo(t)
	hub := mockports.NewStaffRefundHub(t)
	svc := app.NewCustomerFinanceApp(repo, hub, nil, nil, nil)

	requestID := uuid.New()
	repo.On("GetRefundRequest", mock.Anything, requestID).Return(&models.BalanceRefundRequest{
		ID: requestID, CustomerID: uuid.New(),
	}, nil)

	_, err := svc.GetRefundRequest(context.Background(), uuid.New(), requestID)

	assert.ErrorIs(t, err, errcustom.ErrNotFound)
}

func TestListActiveForStaff_Success(t *testing.T) {
	repo := mockports.NewCustomerFinanceRepo(t)
	hub := mockports.NewStaffRefundHub(t)
	svc := app.NewCustomerFinanceApp(repo, hub, nil, nil, nil)

	now := time.Now()
	repo.On("ListActiveRefundRequests", mock.Anything).Return([]models.BalanceRefundRequest{
		{ID: uuid.New(), CustomerID: uuid.New(), AmountBs: 50, Status: models.RefundPending, CreatedAt: now, ExpiresAt: now.Add(time.Hour)},
	}, nil)

	list, err := svc.ListActiveForStaff(context.Background())

	require.NoError(t, err)
	assert.Len(t, list, 1)
	assert.Equal(t, "refund_request_update", list[0].Type)
}

func TestRejectRefund_Success(t *testing.T) {
	repo := mockports.NewCustomerFinanceRepo(t)
	hub := mockports.NewStaffRefundHub(t)
	svc := app.NewCustomerFinanceApp(repo, hub, nil, nil, nil)

	requestID := uuid.New()
	staffID := uuid.New()
	now := time.Now()
	repo.On("RejectPendingWithRestore", mock.Anything, requestID, staffID, "no cumple").Return(true, nil)
	repo.On("GetRefundRequest", mock.Anything, requestID).Return(&models.BalanceRefundRequest{
		ID: requestID, Status: models.RefundRejected, CreatedAt: now, ExpiresAt: now.Add(time.Hour),
	}, nil)
	hub.On("Broadcast", mock.Anything).Return(nil)

	resp, err := svc.RejectRefund(context.Background(), requestID, staffID, "no cumple")

	require.NoError(t, err)
	assert.Equal(t, string(models.RefundRejected), resp.Status)
}

func TestRejectRefund_InvalidStatus(t *testing.T) {
	repo := mockports.NewCustomerFinanceRepo(t)
	hub := mockports.NewStaffRefundHub(t)
	svc := app.NewCustomerFinanceApp(repo, hub, nil, nil, nil)

	requestID := uuid.New()
	staffID := uuid.New()
	repo.On("RejectPendingWithRestore", mock.Anything, requestID, staffID, "").Return(false, nil)

	_, err := svc.RejectRefund(context.Background(), requestID, staffID, "")

	assert.ErrorIs(t, err, errcustom.ErrInvalidStatus)
}

func TestNotifyIncompleteRefunds_PublishesForDueRequests(t *testing.T) {
	repo := mockports.NewCustomerFinanceRepo(t)
	hub := mockports.NewStaffRefundHub(t)
	stream := mockports.NewStreamPublisher(t)
	svc := app.NewCustomerFinanceApp(repo, hub, stream, nil, nil)

	requestID := uuid.New()
	now := time.Now()
	req := models.BalanceRefundRequest{ID: requestID, Status: models.RefundPending, CreatedAt: now.Add(-10 * time.Minute)}

	alertBeforeMatcher := mock.MatchedBy(func(t time.Time) bool { return !t.IsZero() })
	repo.On("ListIncompleteForAlert", mock.Anything, alertBeforeMatcher).Return([]models.BalanceRefundRequest{req}, nil)
	repo.On("MarkAlerted", mock.Anything, requestID, mock.Anything, alertBeforeMatcher).Return(true, nil)
	stream.On("PublisherEvent", mock.Anything, mock.MatchedBy(func(data any) bool {
		ev, ok := data.(models.DataEventBalanceRefundPending)
		return ok && ev.RequestID == requestID.String()
	})).Return(nil)

	err := svc.NotifyIncompleteRefunds(context.Background())

	require.NoError(t, err)
}

func TestNotifyIncompleteRefunds_NoStreamIsNoop(t *testing.T) {
	repo := mockports.NewCustomerFinanceRepo(t)
	hub := mockports.NewStaffRefundHub(t)
	svc := app.NewCustomerFinanceApp(repo, hub, nil, nil, nil)

	err := svc.NotifyIncompleteRefunds(context.Background())

	assert.NoError(t, err)
}
