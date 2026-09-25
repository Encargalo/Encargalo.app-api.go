package app_test

import (
	"context"
	"testing"
	"time"

	bcvmodels "Encargalo.app-api.go/internal/bcv/domain/models"
	mockbcvports "Encargalo.app-api.go/internal/mocks/bcv/domain/ports"
	mockports "Encargalo.app-api.go/internal/mocks/users/domain/ports"
	paymentsModels "Encargalo.app-api.go/internal/payments/domain/models"
	"Encargalo.app-api.go/internal/users/app"
	"Encargalo.app-api.go/internal/users/domain/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGetBalance_TripCreditExposesTripID(t *testing.T) {
	repo := mockports.NewCustomerFinanceRepo(t)
	hub := mockports.NewStaffRefundHub(t)
	bcv := mockbcvports.NewBcvApp(t)
	svc := app.NewCustomerFinanceApp(repo, hub, nil, nil, bcv)

	customerID := uuid.New()
	tripID := uuid.New()
	refType := paymentsModels.ReferenceTypeTrip
	repo.On("GetBalance", mock.Anything, customerID, models.EntityTypeCustomer).
		Return(models.Balance{EntityID: customerID, Amount: 200, Currency: models.CurrencyVES}, nil)
	repo.On("ListTransactions", mock.Anything, customerID, models.EntityTypeCustomer, 1, 10).
		Return([]models.Transaction{{
			ID:            uuid.New(),
			ReferenceID:   &tripID,
			ReferenceType: &refType,
			Amount:        200,
			Currency:      models.CurrencyVES,
			MovementType:  string(models.MovementTypeOrderRejectedRefund),
			CreatedAt:     time.Now(),
		}}, 0, nil)
	bcv.On("GetRate", mock.Anything).Return(&bcvmodels.ExchangeRate{Rate: 40}, nil)

	resp, err := svc.GetBalance(context.Background(), customerID)

	require.NoError(t, err)
	require.Len(t, resp.Transactions, 1)
	require.NotNil(t, resp.Transactions[0].TripID)
	require.Equal(t, tripID.String(), *resp.Transactions[0].TripID)
	require.Nil(t, resp.Transactions[0].OrderID)
}
