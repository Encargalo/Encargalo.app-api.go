package app_test

import (
	"context"
	"testing"
	"time"

	mockports "Encargalo.app-api.go/internal/mocks/users/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/users/app"
	"Encargalo.app-api.go/internal/users/domain/models"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type stubUploader struct {
	url string
	err error
}

func (s stubUploader) DeleteImage(ctx context.Context, secureURL string) error {
	return nil
}

func (s stubUploader) UploadImage(ctx context.Context, imageB64 string) (*uploader.UploadResult, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &uploader.UploadResult{SecureURL: s.url}, nil
}

func TestCompleteRefund_AmountMismatch(t *testing.T) {
	repo := mockports.NewCustomerFinanceRepo(t)
	hub := mockports.NewStaffRefundHub(t)
	svc := app.NewCustomerFinanceApp(repo, hub, nil, stubUploader{url: "https://res.cloudinary.com/x/image/upload/a.jpg"}, nil)

	requestID := uuid.New()
	now := time.Now()
	repo.On("GetRefundRequest", mock.Anything, requestID).Return(&models.BalanceRefundRequest{
		ID:        requestID,
		AmountBs:  52,
		Status:    models.RefundAccepted,
		CreatedAt: now,
		ExpiresAt: now.Add(time.Hour),
	}, nil)

	_, err := svc.CompleteRefund(context.Background(), requestID, "data:image/jpeg;base64,abc", 50)

	assert.ErrorIs(t, err, errcustom.ErrRefundAmountMismatch)
}

func TestCompleteRefund_UploadsAndCompletes(t *testing.T) {
	repo := mockports.NewCustomerFinanceRepo(t)
	hub := mockports.NewStaffRefundHub(t)
	notifier := mockports.NewCustomerNotifier(t)
	proofURL := "https://res.cloudinary.com/demo/image/upload/v1/proof.jpg"
	svc := app.NewCustomerFinanceApp(repo, hub, nil, stubUploader{url: proofURL}, nil)
	svc.SetCustomerNotifier(notifier)

	requestID := uuid.New()
	customerID := uuid.New()
	now := time.Now()
	accepted := &models.BalanceRefundRequest{
		ID:           requestID,
		CustomerID:   customerID,
		AmountBs:     52,
		Status:       models.RefundAccepted,
		PMPhone:      "0414",
		PMBankCode:   "0102",
		PMNationalID: "V1",
		CreatedAt:    now,
		ExpiresAt:    now.Add(time.Hour),
	}
	completed := *accepted
	completed.Status = models.RefundCompleted
	completed.ProofURL = &proofURL

	repo.On("GetRefundRequest", mock.Anything, requestID).Return(accepted, nil).Once()
	repo.On("CompleteAccepted", mock.Anything, requestID, proofURL).Return(true, nil)
	repo.On("GetRefundRequest", mock.Anything, requestID).Return(&completed, nil).Once()
	hub.On("Broadcast", mock.Anything).Return(nil)
	notifier.On("NotifyCustomer", customerID, mock.Anything).Return(nil)

	resp, err := svc.CompleteRefund(context.Background(), requestID, "data:image/jpeg;base64,abc", 52)

	require.NoError(t, err)
	assert.Equal(t, string(models.RefundCompleted), resp.Status)
	require.NotNil(t, resp.ProofURL)
	assert.Equal(t, proofURL, *resp.ProofURL)
}
