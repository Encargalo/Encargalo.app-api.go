package app_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	mockports "Encargalo.app-api.go/internal/mocks/users/domain/ports"
	"Encargalo.app-api.go/internal/users/app"
	"Encargalo.app-api.go/internal/users/domain/dto"
	"Encargalo.app-api.go/internal/users/domain/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestAcceptRefund_NotifiesCustomer(t *testing.T) {
	repo := mockports.NewCustomerFinanceRepo(t)
	hub := mockports.NewStaffRefundHub(t)
	notifier := mockports.NewCustomerNotifier(t)
	svc := app.NewCustomerFinanceApp(repo, hub, nil, nil, nil)
	svc.SetCustomerNotifier(notifier)

	requestID := uuid.New()
	customerID := uuid.New()
	staffID := uuid.New()
	txID := uuid.New()
	now := time.Now()

	pending := &models.BalanceRefundRequest{
		ID:            requestID,
		CustomerID:    customerID,
		TransactionID: txID,
		AmountUSD:     8,
		Status:        models.RefundPending,
		PMPhone:       "04121234567",
		PMBankCode:    "0102",
		PMNationalID:  "V123",
		CreatedAt:     now,
		ExpiresAt:     now.Add(15 * time.Minute),
	}
	accepted := *pending
	accepted.Status = models.RefundAccepted

	repo.On("GetRefundRequest", mock.Anything, requestID).Return(pending, nil).Once()
	repo.On("AcceptPending", mock.Anything, requestID, staffID).Return(true, nil)
	repo.On("GetRefundRequest", mock.Anything, requestID).Return(&accepted, nil).Once()
	hub.On("Broadcast", mock.MatchedBy(func(b dto.RefundRequestBroadcast) bool {
		return b.Type == "refund_request_update" && b.Status == string(models.RefundAccepted) && b.ID == requestID.String()
	})).Return(nil)
	notifier.On("NotifyCustomer", customerID, mock.MatchedBy(func(msg []byte) bool {
		var b dto.RefundRequestBroadcast
		if err := json.Unmarshal(msg, &b); err != nil {
			return false
		}
		return b.Type == "refund_request_update" && b.Status == string(models.RefundAccepted)
	})).Return(nil)

	resp, err := svc.AcceptRefund(context.Background(), requestID, staffID)

	require.NoError(t, err)
	assert.Equal(t, string(models.RefundAccepted), resp.Status)
}
