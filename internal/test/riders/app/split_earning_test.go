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

func TestRecordRideCashCollected_DebitsCashAboveRiderPay(t *testing.T) {
	repo := mocksriders.NewFinanceRepo(t)
	svc := app.NewFinanceApp(repo, nil, nil, nil, nil)

	riderID := uuid.New()
	orderID := uuid.New()
	const delivery = 2.5
	const cashUSD = 4.0
	const rate = 36.0
	deltaBs := money.RoundBs((models.RiderPayUSD(delivery) - cashUSD) * rate)

	repo.On("RecordMovement", mock.Anything, riderID, "rider", deltaBs, 0.0, mock.MatchedBy(func(tx models.Transaction) bool {
		return tx.MovementType == string(models.MovementTypeRideCash)
	})).Return(0.0, nil)

	err := svc.RecordRideCashCollected(context.Background(), riderID, orderID, delivery, cashUSD, rate)

	require.NoError(t, err)
	require.Equal(t, -63.0, deltaBs)
}
