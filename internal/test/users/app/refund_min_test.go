package app_test

import (
	"context"
	"testing"
	"time"

	mockports "Encargalo.app-api.go/internal/mocks/users/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/users/app"
	"Encargalo.app-api.go/internal/users/domain/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCreateRefundRequest_ExactOneUSDNotEligible(t *testing.T) {
	repo := mockports.NewCustomerFinanceRepo(t)
	hub := mockports.NewStaffRefundHub(t)
	svc := app.NewCustomerFinanceApp(repo, hub, nil, nil, nil)

	customerID := uuid.New()
	txID := uuid.New()
	rate := 40.0
	oneUSDInBs := models.RefundMinAmountBs(rate)

	repo.On("GetPagoMovil", mock.Anything, customerID).Return(&models.CustomerPagoMovil{
		CustomerID: customerID,
		Phone:      "+58412",
		BankCode:   "0102",
		NationalID: "V1",
	}, nil)
	repo.On("GetTransaction", mock.Anything, txID).Return(&models.Transaction{
		ID:             txID,
		EntityID:       customerID,
		EntityType:     models.EntityTypeCustomer,
		Amount:         oneUSDInBs,
		MovementType:   string(models.MovementTypePaymentOverage),
		RefundablePMBs: &oneUSDInBs,
		BcvRate:        &rate,
		CreatedAt:      time.Now(),
	}, nil)

	_, err := svc.CreateRefundRequest(context.Background(), customerID, txID)

	assert.ErrorIs(t, err, errcustom.ErrRefundNotEligible)
}

func TestCreateRefundRequest_AboveOneUSDEligible(t *testing.T) {
	repo := mockports.NewCustomerFinanceRepo(t)
	hub := mockports.NewStaffRefundHub(t)
	svc := app.NewCustomerFinanceApp(repo, hub, nil, nil, nil)

	customerID := uuid.New()
	txID := uuid.New()
	rate := 40.0
	amountBs := models.RefundMinAmountBs(rate) + 0.01

	repo.On("GetPagoMovil", mock.Anything, customerID).Return(&models.CustomerPagoMovil{
		CustomerID: customerID,
		Phone:      "+58412",
		BankCode:   "0102",
		NationalID: "V1",
	}, nil)
	repo.On("GetTransaction", mock.Anything, txID).Return(&models.Transaction{
		ID:             txID,
		EntityID:       customerID,
		EntityType:     models.EntityTypeCustomer,
		Amount:         amountBs,
		MovementType:   string(models.MovementTypePaymentOverage),
		RefundablePMBs: &amountBs,
		BcvRate:        &rate,
		CreatedAt:      time.Now(),
	}, nil)
	repo.On("HasActiveRefundForTransaction", mock.Anything, txID).Return(false, nil)
	repo.On("CreateRefundRequestWithDebit", mock.Anything, mock.MatchedBy(func(req *models.BalanceRefundRequest) bool {
		return req.AmountBs == amountBs && req.Status == models.RefundPending
	})).Return(nil)
	hub.On("Broadcast", mock.Anything).Return(nil)

	resp, err := svc.CreateRefundRequest(context.Background(), customerID, txID)

	assert.NoError(t, err)
	assert.Equal(t, amountBs, resp.AmountBs)
}

func TestCreateRefundRequest_NoFrozenRateNotEligible(t *testing.T) {
	repo := mockports.NewCustomerFinanceRepo(t)
	hub := mockports.NewStaffRefundHub(t)
	svc := app.NewCustomerFinanceApp(repo, hub, nil, nil, nil)

	customerID := uuid.New()
	txID := uuid.New()
	amountBs := 200.0

	repo.On("GetPagoMovil", mock.Anything, customerID).Return(&models.CustomerPagoMovil{
		CustomerID: customerID,
		Phone:      "+58412",
		BankCode:   "0102",
		NationalID: "V1",
	}, nil)
	repo.On("GetTransaction", mock.Anything, txID).Return(&models.Transaction{
		ID:             txID,
		EntityID:       customerID,
		EntityType:     models.EntityTypeCustomer,
		Amount:         amountBs,
		MovementType:   string(models.MovementTypePaymentOverage),
		RefundablePMBs: &amountBs,
		CreatedAt:      time.Now(),
	}, nil)

	_, err := svc.CreateRefundRequest(context.Background(), customerID, txID)

	assert.ErrorIs(t, err, errcustom.ErrRefundNotEligible)
}
