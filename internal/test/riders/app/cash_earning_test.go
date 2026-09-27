package app_test

import (
	"context"
	"testing"

	mocksriders "Encargalo.app-api.go/internal/mocks/riders/domain/ports"
	"Encargalo.app-api.go/internal/pkg/money"
	"Encargalo.app-api.go/internal/riders/app"
	"Encargalo.app-api.go/internal/riders/domain/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestRecordRideEarning_CashDebitsRemainderAfterRiderPay(t *testing.T) {
	repo := mocksriders.NewFinanceRepo(t)
	svc := app.NewFinanceApp(repo, nil)

	riderID := uuid.New()
	orderID := uuid.New()
	// Cobra $10. Su pago es el 80% de un envío de $2.50 = $2. Nos debe $8.
	const total = 10.0
	const delivery = 2.5
	const rate = 36.0
	owedBs := money.RoundBs(-models.CashOwedUSD(total, models.RiderPayUSD(delivery)) * rate)

	repo.On("RecordMovement", mock.Anything, riderID, "rider", owedBs, 0.0, mock.MatchedBy(func(tx models.Transaction) bool {
		return tx.MovementType == string(models.MovementTypeRideCash) &&
			tx.PaymentMethod != nil && *tx.PaymentMethod == models.PaymentMethodEfectivo
	})).Return(0.0, nil)

	err := svc.RecordRideEarning(context.Background(), riderID, orderID, delivery, total, rate, models.PaymentMethodEfectivo)

	require.NoError(t, err)
	require.Equal(t, 8.0, models.CashOwedUSD(total, models.RiderPayUSD(delivery)))
	require.Equal(t, -288.0, owedBs)
}

func TestRecordTripCash_DebitsPlatformShare(t *testing.T) {
	repo := mocksriders.NewFinanceRepo(t)
	svc := app.NewFinanceApp(repo, nil)

	riderID := uuid.New()
	tripID := uuid.New()
	const fare = 10.0
	const rate = 36.0
	owedBs := money.RoundBs(-models.CashOwedUSD(fare, models.RiderPayUSD(fare)) * rate)

	repo.On("RecordMovement", mock.Anything, riderID, "rider", owedBs, 0.0, mock.MatchedBy(func(tx models.Transaction) bool {
		return tx.MovementType == string(models.MovementTypeTripCash) &&
			tx.TripID != nil && *tx.TripID == tripID
	})).Return(0.0, nil)

	err := svc.RecordTripCash(context.Background(), riderID, tripID, fare, rate)

	require.NoError(t, err)
	require.Equal(t, -72.0, owedBs)
}
