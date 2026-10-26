package app_test

import (
	"context"
	"testing"
	"time"

	bcvmodels "Encargalo.app-api.go/internal/bcv/domain/models"
	financeModels "Encargalo.app-api.go/internal/finance/domain/models"
	financePorts "Encargalo.app-api.go/internal/finance/domain/ports"
	mockbcvports "Encargalo.app-api.go/internal/mocks/bcv/domain/ports"
	mocksriders "Encargalo.app-api.go/internal/mocks/riders/domain/ports"
	"Encargalo.app-api.go/internal/riders/app"
	"Encargalo.app-api.go/internal/riders/domain/models"
	"Encargalo.app-api.go/internal/shared/errcustom"
	usersModels "Encargalo.app-api.go/internal/users/domain/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestRecordRideEarning_CreditsNinetyPercentInBs(t *testing.T) {
	repo := mocksriders.NewFinanceRepo(t)
	svc := app.NewFinanceApp(repo, mockbcvports.NewBcvApp(t), nil, nil, nil)

	riderID := uuid.New()
	orderID := uuid.New()

	repo.On("RecordMovement", mock.Anything, riderID, "rider", 40*0.90*36.0, 0.0, mock.MatchedBy(func(tx models.Transaction) bool {
		return tx.MovementType == string(models.MovementTypeRideBank) && tx.OrderID != nil && *tx.OrderID == orderID
	})).Return(0.0, nil)

	err := svc.RecordRideEarning(context.Background(), riderID, orderID, 40, 100, 36.0, "PagoMovil")

	require.NoError(t, err)
}

func TestRecordRideEarning_NoRate_Fails(t *testing.T) {
	repo := mocksriders.NewFinanceRepo(t)
	svc := app.NewFinanceApp(repo, mockbcvports.NewBcvApp(t), nil, nil, nil)

	err := svc.RecordRideEarning(context.Background(), uuid.New(), uuid.New(), 40, 100, 0, "PagoMovil")

	assert.ErrorIs(t, err, errcustom.ErrBcvRateUnavailable)
}

func TestRecordTripEarning_CreditsNinetyPercentInBs(t *testing.T) {
	repo := mocksriders.NewFinanceRepo(t)
	svc := app.NewFinanceApp(repo, mockbcvports.NewBcvApp(t), nil, nil, nil)

	riderID := uuid.New()
	tripID := uuid.New()

	repo.On("RecordMovement", mock.Anything, riderID, "rider", 10*0.90*36.0, 0.0, mock.MatchedBy(func(tx models.Transaction) bool {
		return tx.MovementType == string(models.MovementTypeTripBank) &&
			tx.TripID != nil && *tx.TripID == tripID &&
			tx.ServiceType != nil && *tx.ServiceType == models.ServiceTypeTrip
	})).Return(0.0, nil)

	err := svc.RecordTripEarning(context.Background(), riderID, tripID, 10, 36.0)

	require.NoError(t, err)
}

func TestRecordTripEarning_NoRate_Fails(t *testing.T) {
	repo := mocksriders.NewFinanceRepo(t)
	svc := app.NewFinanceApp(repo, mockbcvports.NewBcvApp(t), nil, nil, nil)

	err := svc.RecordTripEarning(context.Background(), uuid.New(), uuid.New(), 10, 0)

	assert.ErrorIs(t, err, errcustom.ErrBcvRateUnavailable)
}

func TestGetBalance_BuildsResponseWithZoneAndTransactions(t *testing.T) {
	repo := mocksriders.NewFinanceRepo(t)
	bcv := mockbcvports.NewBcvApp(t)
	svc := app.NewFinanceApp(repo, bcv, nil, nil, nil)

	riderID := uuid.New()
	repo.On("GetBalance", mock.Anything, riderID, "rider").Return(models.Balance{Amount: 600, Currency: models.CurrencyVES}, nil)
	repo.On("ListRecentTransactions", mock.Anything, riderID, "rider", 10).Return([]models.Transaction{{ID: uuid.New(), MovementType: string(models.MovementTypeRideBank), Currency: models.CurrencyVES, CreatedAt: time.Now()}}, nil)
	bcv.On("GetRate", mock.Anything).Return(&bcvmodels.ExchangeRate{Rate: 40}, nil)

	resp, err := svc.GetBalance(context.Background(), riderID)

	require.NoError(t, err)
	assert.Equal(t, 600.0, resp.BalanceBs)
	assert.Equal(t, 40.0, resp.BcvRate)
	assert.Equal(t, 120.0, resp.WithdrawalMinBs)
	assert.Equal(t, models.ZoneWithdrawalAvailable, resp.Zone)
	assert.Len(t, resp.Transactions, 1)
}

func TestGetBalance_RepoError(t *testing.T) {
	repo := mocksriders.NewFinanceRepo(t)
	svc := app.NewFinanceApp(repo, mockbcvports.NewBcvApp(t), nil, nil, nil)

	riderID := uuid.New()
	repo.On("GetBalance", mock.Anything, riderID, "rider").Return(models.Balance{}, errcustom.ErrUnexpectedError)

	_, err := svc.GetBalance(context.Background(), riderID)

	assert.ErrorIs(t, err, errcustom.ErrUnexpectedError)
}

func TestRequestWithdrawal_CreatesPendingRequest(t *testing.T) {
	repo := mocksriders.NewFinanceRepo(t)
	bcv := mockbcvports.NewBcvApp(t)
	withdrawals := mocksriders.NewWithdrawalStore(t)
	payments := mocksriders.NewPaymentDetailsStore(t)
	events := mocksriders.NewWithdrawalPublisher(t)
	svc := app.NewFinanceApp(repo, bcv, withdrawals, payments, events)

	riderID := uuid.New()
	requestID := uuid.New()
	withdrawals.On("LastCompletedAt", mock.Anything, riderID, "rider").Return(nil, nil)
	withdrawals.On("GetActive", mock.Anything, riderID, "rider").Return(nil, nil)
	bcv.On("GetRate", mock.Anything).Return(&bcvmodels.ExchangeRate{Rate: 40}, nil)
	payments.On("GetPagoMovil", mock.Anything, riderID).
		Return(&usersModels.CustomerPagoMovil{Phone: "04121234567", BankCode: "0102", NationalID: "V12345678"}, nil)
	// El mínimo viaja en Bs a la tasa vigente: $3 * 40.
	withdrawals.On("Create", mock.Anything, mock.MatchedBy(func(p financePorts.CreateWithdrawalParams) bool {
		return p.EntityID == riderID && p.EntityType == "rider" && p.MinAmount == 120.0 && p.PMPhone == "04121234567"
	})).Return(financeModels.WithdrawalRequest{
		ID:        requestID,
		EntityID:  riderID,
		Amount:    800,
		Status:    financeModels.WithdrawalPending,
		CreatedAt: time.Now(),
	}, nil)
	events.On("PublishWithdrawalRequested", mock.Anything, mock.Anything).Return(nil)

	resp, err := svc.RequestWithdrawal(context.Background(), riderID)

	require.NoError(t, err)
	assert.Equal(t, requestID.String(), resp.ID)
	assert.Equal(t, 800.0, resp.AmountBs)
	assert.Equal(t, string(financeModels.WithdrawalPending), resp.Status)
}

// El aviso es conveniencia: si falla la publicación, la solicitud ya existe.
func TestRequestWithdrawal_SucceedsWhenPublishFails(t *testing.T) {
	repo := mocksriders.NewFinanceRepo(t)
	bcv := mockbcvports.NewBcvApp(t)
	withdrawals := mocksriders.NewWithdrawalStore(t)
	payments := mocksriders.NewPaymentDetailsStore(t)
	events := mocksriders.NewWithdrawalPublisher(t)
	svc := app.NewFinanceApp(repo, bcv, withdrawals, payments, events)

	riderID := uuid.New()
	withdrawals.On("LastCompletedAt", mock.Anything, riderID, "rider").Return(nil, nil)
	withdrawals.On("GetActive", mock.Anything, riderID, "rider").Return(nil, nil)
	bcv.On("GetRate", mock.Anything).Return(&bcvmodels.ExchangeRate{Rate: 40}, nil)
	payments.On("GetPagoMovil", mock.Anything, riderID).
		Return(&usersModels.CustomerPagoMovil{Phone: "04121234567"}, nil)
	withdrawals.On("Create", mock.Anything, mock.Anything).
		Return(financeModels.WithdrawalRequest{ID: uuid.New(), Amount: 800, Status: financeModels.WithdrawalPending, CreatedAt: time.Now()}, nil)
	events.On("PublishWithdrawalRequested", mock.Anything, mock.Anything).Return(errcustom.ErrUnexpectedError)

	_, err := svc.RequestWithdrawal(context.Background(), riderID)

	require.NoError(t, err)
}

// El cooldown se mide contra el último retiro PAGADO.
func TestRequestWithdrawal_CooldownFromLastPayout(t *testing.T) {
	withdrawals := mocksriders.NewWithdrawalStore(t)
	svc := app.NewFinanceApp(mocksriders.NewFinanceRepo(t), mockbcvports.NewBcvApp(t), withdrawals, nil, nil)

	riderID := uuid.New()
	paid := time.Now().Add(-1 * time.Hour)
	withdrawals.On("LastCompletedAt", mock.Anything, riderID, "rider").Return(&paid, nil)

	_, err := svc.RequestWithdrawal(context.Background(), riderID)

	assert.ErrorIs(t, err, errcustom.ErrWithdrawalCooldown)
}

// Un retiro pagado hace más de 24h no bloquea.
func TestRequestWithdrawal_CooldownExpired(t *testing.T) {
	bcv := mockbcvports.NewBcvApp(t)
	withdrawals := mocksriders.NewWithdrawalStore(t)
	payments := mocksriders.NewPaymentDetailsStore(t)
	svc := app.NewFinanceApp(mocksriders.NewFinanceRepo(t), bcv, withdrawals, payments, nil)

	riderID := uuid.New()
	paid := time.Now().Add(-25 * time.Hour)
	withdrawals.On("LastCompletedAt", mock.Anything, riderID, "rider").Return(&paid, nil)
	withdrawals.On("GetActive", mock.Anything, riderID, "rider").Return(nil, nil)
	bcv.On("GetRate", mock.Anything).Return(&bcvmodels.ExchangeRate{Rate: 40}, nil)
	payments.On("GetPagoMovil", mock.Anything, riderID).
		Return(&usersModels.CustomerPagoMovil{Phone: "04121234567"}, nil)
	withdrawals.On("Create", mock.Anything, mock.Anything).
		Return(financeModels.WithdrawalRequest{ID: uuid.New(), Amount: 800, Status: financeModels.WithdrawalPending, CreatedAt: time.Now()}, nil)

	_, err := svc.RequestWithdrawal(context.Background(), riderID)

	require.NoError(t, err)
}

func TestRequestWithdrawal_AlreadyRequested(t *testing.T) {
	withdrawals := mocksriders.NewWithdrawalStore(t)
	svc := app.NewFinanceApp(mocksriders.NewFinanceRepo(t), mockbcvports.NewBcvApp(t), withdrawals, nil, nil)

	riderID := uuid.New()
	withdrawals.On("LastCompletedAt", mock.Anything, riderID, "rider").Return(nil, nil)
	withdrawals.On("GetActive", mock.Anything, riderID, "rider").
		Return(&financeModels.WithdrawalRequest{ID: uuid.New(), Status: financeModels.WithdrawalPending}, nil)

	_, err := svc.RequestWithdrawal(context.Background(), riderID)

	assert.ErrorIs(t, err, errcustom.ErrWithdrawalAlreadyRequested)
}

// Sin Pago Móvil no hay destino al que transferir, y no se retiene saldo.
func TestRequestWithdrawal_PagoMovilRequired(t *testing.T) {
	bcv := mockbcvports.NewBcvApp(t)
	withdrawals := mocksriders.NewWithdrawalStore(t)
	payments := mocksriders.NewPaymentDetailsStore(t)
	svc := app.NewFinanceApp(mocksriders.NewFinanceRepo(t), bcv, withdrawals, payments, nil)

	riderID := uuid.New()
	withdrawals.On("LastCompletedAt", mock.Anything, riderID, "rider").Return(nil, nil)
	withdrawals.On("GetActive", mock.Anything, riderID, "rider").Return(nil, nil)
	bcv.On("GetRate", mock.Anything).Return(&bcvmodels.ExchangeRate{Rate: 40}, nil)
	payments.On("GetPagoMovil", mock.Anything, riderID).Return(nil, errcustom.ErrNotFound)

	_, err := svc.RequestWithdrawal(context.Background(), riderID)

	assert.ErrorIs(t, err, errcustom.ErrPagoMovilRequired)
}

func TestRequestWithdrawal_InsufficientBalance(t *testing.T) {
	bcv := mockbcvports.NewBcvApp(t)
	withdrawals := mocksriders.NewWithdrawalStore(t)
	payments := mocksriders.NewPaymentDetailsStore(t)
	svc := app.NewFinanceApp(mocksriders.NewFinanceRepo(t), bcv, withdrawals, payments, nil)

	riderID := uuid.New()
	withdrawals.On("LastCompletedAt", mock.Anything, riderID, "rider").Return(nil, nil)
	withdrawals.On("GetActive", mock.Anything, riderID, "rider").Return(nil, nil)
	bcv.On("GetRate", mock.Anything).Return(&bcvmodels.ExchangeRate{Rate: 40}, nil)
	payments.On("GetPagoMovil", mock.Anything, riderID).
		Return(&usersModels.CustomerPagoMovil{Phone: "04121234567"}, nil)
	withdrawals.On("Create", mock.Anything, mock.Anything).
		Return(financeModels.WithdrawalRequest{}, errcustom.ErrInsufficientBalance)

	_, err := svc.RequestWithdrawal(context.Background(), riderID)

	assert.ErrorIs(t, err, errcustom.ErrInsufficientBalance)
}

func TestRequestWithdrawal_RateUnavailable(t *testing.T) {
	bcv := mockbcvports.NewBcvApp(t)
	withdrawals := mocksriders.NewWithdrawalStore(t)
	svc := app.NewFinanceApp(mocksriders.NewFinanceRepo(t), bcv, withdrawals, nil, nil)

	riderID := uuid.New()
	withdrawals.On("LastCompletedAt", mock.Anything, riderID, "rider").Return(nil, nil)
	withdrawals.On("GetActive", mock.Anything, riderID, "rider").Return(nil, nil)
	bcv.On("GetRate", mock.Anything).Return(nil, errcustom.ErrUnexpectedError)

	_, err := svc.RequestWithdrawal(context.Background(), riderID)

	assert.ErrorIs(t, err, errcustom.ErrBcvRateUnavailable)
}

func TestListTransactions_CapsLimitAt50(t *testing.T) {
	repo := mocksriders.NewFinanceRepo(t)
	bcv := mockbcvports.NewBcvApp(t)
	svc := app.NewFinanceApp(repo, bcv, nil, nil, nil)

	riderID := uuid.New()
	repo.On("ListTransactions", mock.Anything, riderID, "rider", 1, 50).Return([]models.Transaction{}, 0, nil)
	bcv.On("GetRate", mock.Anything).Return(&bcvmodels.ExchangeRate{Rate: 40}, nil)

	resp, err := svc.ListTransactions(context.Background(), riderID, 1, 999)

	require.NoError(t, err)
	assert.Equal(t, 50, resp.Limit)
}

func TestListTransactions_RepoError(t *testing.T) {
	repo := mocksriders.NewFinanceRepo(t)
	svc := app.NewFinanceApp(repo, mockbcvports.NewBcvApp(t), nil, nil, nil)

	riderID := uuid.New()
	repo.On("ListTransactions", mock.Anything, riderID, "rider", 1, 10).Return(nil, 0, errcustom.ErrUnexpectedError)

	_, err := svc.ListTransactions(context.Background(), riderID, 1, 10)

	assert.ErrorIs(t, err, errcustom.ErrUnexpectedError)
}

func TestRiderEarning_NinetyPercentOfDeliveryFee(t *testing.T) {
	assert.InDelta(t, 36.0, app.RiderEarning(40), 0.0001)
}
