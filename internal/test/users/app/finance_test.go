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

func TestCreateRefundRequest_OnlyPMPortion(t *testing.T) {
	repo := mockports.NewCustomerFinanceRepo(t)
	hub := mockports.NewStaffRefundHub(t)
	svc := app.NewCustomerFinanceApp(repo, hub, nil, nil, nil)

	customerID := uuid.New()
	txID := uuid.New()
	orderID := uuid.New()
	refundable := 5.0

	repo.On("GetPagoMovil", mock.Anything, customerID).Return(&models.CustomerPagoMovil{
		CustomerID: customerID,
		Phone:      "+58412",
		BankCode:   "0102",
		NationalID: "V1",
	}, nil)
	repo.On("GetTransaction", mock.Anything, txID).Return(&models.Transaction{
		ID:              txID,
		EntityID:        customerID,
		EntityType:      models.EntityTypeCustomer,
		OrderID:         &orderID,
		Amount:          10,
		MovementType:    string(models.MovementTypeOrderRejectedRefund),
		RefundablePMUSD: &refundable,
		CreatedAt:       time.Now(),
	}, nil)
	repo.On("HasActiveRefundForTransaction", mock.Anything, txID).Return(false, nil)
	repo.On("CreateRefundRequestWithDebit", mock.Anything, mock.MatchedBy(func(req *models.BalanceRefundRequest) bool {
		return req.AmountUSD == 5.0 && req.Status == models.RefundPending
	})).Return(nil)
	hub.On("Broadcast", mock.Anything).Return(nil)

	resp, err := svc.CreateRefundRequest(context.Background(), customerID, txID)

	assert.NoError(t, err)
	assert.Equal(t, 5.0, resp.AmountUSD)
	assert.Equal(t, string(models.RefundPending), resp.Status)
}

func TestCreateRefundRequest_HundredPercentBalanceNotEligible(t *testing.T) {
	repo := mockports.NewCustomerFinanceRepo(t)
	hub := mockports.NewStaffRefundHub(t)
	svc := app.NewCustomerFinanceApp(repo, hub, nil, nil, nil)

	customerID := uuid.New()
	txID := uuid.New()
	zero := 0.0

	repo.On("GetPagoMovil", mock.Anything, customerID).Return(&models.CustomerPagoMovil{
		CustomerID: customerID,
		Phone:      "+58412",
		BankCode:   "0102",
		NationalID: "V1",
	}, nil)
	repo.On("GetTransaction", mock.Anything, txID).Return(&models.Transaction{
		ID:              txID,
		EntityID:        customerID,
		EntityType:      models.EntityTypeCustomer,
		Amount:          10,
		MovementType:    string(models.MovementTypeOrderRejectedRefund),
		RefundablePMUSD: &zero,
		CreatedAt:       time.Now(),
	}, nil)

	_, err := svc.CreateRefundRequest(context.Background(), customerID, txID)

	assert.ErrorIs(t, err, errcustom.ErrRefundNotEligible)
}

func TestCreateRefundRequest_RequiresPagoMovil(t *testing.T) {
	repo := mockports.NewCustomerFinanceRepo(t)
	hub := mockports.NewStaffRefundHub(t)
	svc := app.NewCustomerFinanceApp(repo, hub, nil, nil, nil)

	customerID := uuid.New()
	txID := uuid.New()

	repo.On("GetPagoMovil", mock.Anything, customerID).Return(nil, errcustom.ErrNotFound)

	_, err := svc.CreateRefundRequest(context.Background(), customerID, txID)

	assert.ErrorIs(t, err, errcustom.ErrPagoMovilRequired)
}
