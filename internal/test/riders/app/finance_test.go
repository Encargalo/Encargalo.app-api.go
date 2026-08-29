package app_test

import (
	"context"
	"testing"
	"time"

	bcvmodels "Encargalo.app-api.go/internal/bcv/domain/models"
	mockbcvports "Encargalo.app-api.go/internal/mocks/bcv/domain/ports"
	mocksriders "Encargalo.app-api.go/internal/mocks/riders/domain/ports"
	"Encargalo.app-api.go/internal/riders/app"
	"Encargalo.app-api.go/internal/riders/domain/models"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestRecordRideEarning_CreditsEightyPercentInBs(t *testing.T) {
	repo := mocksriders.NewFinanceRepo(t)
	svc := app.NewFinanceApp(repo, mockbcvports.NewBcvApp(t))

	riderID := uuid.New()
	orderID := uuid.New()

	repo.On("RecordMovement", mock.Anything, riderID, "rider", 40*0.80*36.0, 0.0, mock.MatchedBy(func(tx models.Transaction) bool {
		return tx.MovementType == string(models.MovementTypeRideBank) && tx.OrderID != nil && *tx.OrderID == orderID
	})).Return(0.0, nil)

	err := svc.RecordRideEarning(context.Background(), riderID, orderID, 40, 100, 36.0, "PagoMovil")

	require.NoError(t, err)
}

func TestRecordRideEarning_NoRate_Fails(t *testing.T) {
	repo := mocksriders.NewFinanceRepo(t)
	svc := app.NewFinanceApp(repo, mockbcvports.NewBcvApp(t))

	err := svc.RecordRideEarning(context.Background(), uuid.New(), uuid.New(), 40, 100, 0, "PagoMovil")

	assert.ErrorIs(t, err, errcustom.ErrBcvRateUnavailable)
}

func TestRecordTripEarning_CreditsEightyPercentInBs(t *testing.T) {
	repo := mocksriders.NewFinanceRepo(t)
	svc := app.NewFinanceApp(repo, mockbcvports.NewBcvApp(t))

	riderID := uuid.New()
	tripID := uuid.New()

	repo.On("RecordMovement", mock.Anything, riderID, "rider", 10*0.80*36.0, 0.0, mock.MatchedBy(func(tx models.Transaction) bool {
		return tx.MovementType == string(models.MovementTypeTripBank) &&
			tx.TripID != nil && *tx.TripID == tripID &&
			tx.ServiceType != nil && *tx.ServiceType == models.ServiceTypeTrip
	})).Return(0.0, nil)

	err := svc.RecordTripEarning(context.Background(), riderID, tripID, 10, 36.0)

	require.NoError(t, err)
}

func TestRecordTripEarning_NoRate_Fails(t *testing.T) {
	repo := mocksriders.NewFinanceRepo(t)
	svc := app.NewFinanceApp(repo, mockbcvports.NewBcvApp(t))

	err := svc.RecordTripEarning(context.Background(), uuid.New(), uuid.New(), 10, 0)

	assert.ErrorIs(t, err, errcustom.ErrBcvRateUnavailable)
}

func TestGetBalance_BuildsResponseWithZoneAndTransactions(t *testing.T) {
	repo := mocksriders.NewFinanceRepo(t)
	bcv := mockbcvports.NewBcvApp(t)
	svc := app.NewFinanceApp(repo, bcv)

	riderID := uuid.New()
	repo.On("GetBalance", mock.Anything, riderID, "rider").Return(models.Balance{Amount: 600, Currency: models.CurrencyVES}, nil)
	repo.On("ListTransactions", mock.Anything, riderID, "rider", 1, 10).Return([]models.Transaction{{ID: uuid.New(), MovementType: string(models.MovementTypeRideBank), Currency: models.CurrencyVES, CreatedAt: time.Now()}}, 1, nil)
	bcv.On("GetRate", mock.Anything).Return(&bcvmodels.ExchangeRate{Rate: 40}, nil)

	resp, err := svc.GetBalance(context.Background(), riderID)

	require.NoError(t, err)
	assert.Equal(t, 600.0, resp.BalanceBs)
	assert.Equal(t, 40.0, resp.BcvRate)
	assert.Equal(t, 600.0, resp.WithdrawalMinBs)
	assert.Equal(t, models.ZoneWithdrawalAvailable, resp.Zone)
	assert.Len(t, resp.Transactions, 1)
}

func TestGetBalance_RepoError(t *testing.T) {
	repo := mocksriders.NewFinanceRepo(t)
	svc := app.NewFinanceApp(repo, mockbcvports.NewBcvApp(t))

	riderID := uuid.New()
	repo.On("GetBalance", mock.Anything, riderID, "rider").Return(models.Balance{}, errcustom.ErrUnexpectedError)

	_, err := svc.GetBalance(context.Background(), riderID)

	assert.ErrorIs(t, err, errcustom.ErrUnexpectedError)
}

func TestRequestWithdrawal_Success(t *testing.T) {
	repo := mocksriders.NewFinanceRepo(t)
	bcv := mockbcvports.NewBcvApp(t)
	svc := app.NewFinanceApp(repo, bcv)

	riderID := uuid.New()
	repo.On("GetLastWithdrawal", mock.Anything, riderID, "rider").Return(nil, nil)
	bcv.On("GetRate", mock.Anything).Return(&bcvmodels.ExchangeRate{Rate: 40}, nil)
	repo.On("GetBalance", mock.Anything, riderID, "rider").Return(models.Balance{Amount: 800, Currency: models.CurrencyVES}, nil)
	repo.On("RecordMovement", mock.Anything, riderID, "rider", -800.0, 600.0, mock.MatchedBy(func(tx models.Transaction) bool {
		return tx.MovementType == string(models.MovementTypeWithdraw)
	})).Return(0.0, nil)

	resp, err := svc.RequestWithdrawal(context.Background(), riderID)

	require.NoError(t, err)
	assert.Equal(t, 800.0, resp.AmountWithdrawn)
}

func TestRequestWithdrawal_Cooldown(t *testing.T) {
	repo := mocksriders.NewFinanceRepo(t)
	svc := app.NewFinanceApp(repo, mockbcvports.NewBcvApp(t))

	riderID := uuid.New()
	last := time.Now().Add(-1 * time.Hour)
	repo.On("GetLastWithdrawal", mock.Anything, riderID, "rider").Return(&last, nil)

	_, err := svc.RequestWithdrawal(context.Background(), riderID)

	assert.ErrorIs(t, err, errcustom.ErrWithdrawalCooldown)
}

func TestRequestWithdrawal_RateUnavailable(t *testing.T) {
	repo := mocksriders.NewFinanceRepo(t)
	bcv := mockbcvports.NewBcvApp(t)
	svc := app.NewFinanceApp(repo, bcv)

	riderID := uuid.New()
	repo.On("GetLastWithdrawal", mock.Anything, riderID, "rider").Return(nil, nil)
	bcv.On("GetRate", mock.Anything).Return(nil, errcustom.ErrUnexpectedError)

	_, err := svc.RequestWithdrawal(context.Background(), riderID)

	assert.ErrorIs(t, err, errcustom.ErrBcvRateUnavailable)
}

func TestListTransactions_CapsLimitAt50(t *testing.T) {
	repo := mocksriders.NewFinanceRepo(t)
	bcv := mockbcvports.NewBcvApp(t)
	svc := app.NewFinanceApp(repo, bcv)

	riderID := uuid.New()
	repo.On("ListTransactions", mock.Anything, riderID, "rider", 1, 50).Return([]models.Transaction{}, 0, nil)
	bcv.On("GetRate", mock.Anything).Return(&bcvmodels.ExchangeRate{Rate: 40}, nil)

	resp, err := svc.ListTransactions(context.Background(), riderID, 1, 999)

	require.NoError(t, err)
	assert.Equal(t, 50, resp.Limit)
}

func TestListTransactions_RepoError(t *testing.T) {
	repo := mocksriders.NewFinanceRepo(t)
	svc := app.NewFinanceApp(repo, mockbcvports.NewBcvApp(t))

	riderID := uuid.New()
	repo.On("ListTransactions", mock.Anything, riderID, "rider", 1, 10).Return(nil, 0, errcustom.ErrUnexpectedError)

	_, err := svc.ListTransactions(context.Background(), riderID, 1, 10)

	assert.ErrorIs(t, err, errcustom.ErrUnexpectedError)
}

func TestRiderEarning_EightyPercentOfDeliveryFee(t *testing.T) {
	assert.InDelta(t, 32.0, app.RiderEarning(40), 0.0001)
}
