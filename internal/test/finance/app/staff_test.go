package app_test

import (
	"context"
	"testing"

	bcvmodels "Encargalo.app-api.go/internal/bcv/domain/models"
	"Encargalo.app-api.go/internal/finance/app"
	financeModels "Encargalo.app-api.go/internal/finance/domain/models"
	mockbcv "Encargalo.app-api.go/internal/mocks/bcv/domain/ports"
	mockfinance "Encargalo.app-api.go/internal/mocks/finance/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type stubUploader struct{}

func (stubUploader) DeleteImage(ctx context.Context, secureURL string) error { return nil }
func (stubUploader) UploadImage(ctx context.Context, imageB64 string) (*uploader.UploadResult, error) {
	return &uploader.UploadResult{SecureURL: "https://res.cloudinary.com/x/a.jpg"}, nil
}

func TestRecordPayout_ShopConvertsBsToUSD(t *testing.T) {
	repo := mockfinance.NewStaffFinanceRepo(t)
	bcv := mockbcv.NewBcvApp(t)
	bcv.On("GetRate", mock.Anything).Return(&bcvmodels.ExchangeRate{Rate: 52}, nil)

	svc := app.NewStaffFinanceApp(repo, bcv, stubUploader{})

	entityID := uuid.New()
	staffID := uuid.New()

	repo.On("GetBalance", mock.Anything, entityID, financeModels.EntityTypeShop).
		Return(financeModels.Balance{Amount: 320, Currency: financeModels.CurrencyUSD}, nil)
	repo.On("RecordMovement", mock.Anything, entityID, financeModels.EntityTypeShop, -100.0, 0.0, mock.AnythingOfType("models.Transaction")).
		Return(220.0, nil)

	resp, err := svc.RecordPayout(context.Background(), entityID, staffID, financeModels.EntityTypeShop, 5200, "pago", "data:image/jpeg;base64,abc")
	require.NoError(t, err)
	require.Equal(t, -100.0, resp.AmountUSD)
	require.Equal(t, 220.0, resp.BalanceUSD)
}

func TestRecordPayout_RiderInsufficientBalance(t *testing.T) {
	repo := mockfinance.NewStaffFinanceRepo(t)
	bcv := mockbcv.NewBcvApp(t)
	bcv.On("GetRate", mock.Anything).Return(&bcvmodels.ExchangeRate{Rate: 52}, nil)

	svc := app.NewStaffFinanceApp(repo, bcv, stubUploader{})

	entityID := uuid.New()
	repo.On("GetBalance", mock.Anything, entityID, financeModels.EntityTypeRider).
		Return(financeModels.Balance{Amount: 1000, Currency: financeModels.CurrencyVES}, nil)

	_, err := svc.RecordPayout(context.Background(), entityID, uuid.New(), financeModels.EntityTypeRider, 5200, "", "data:image/jpeg;base64,abc")
	require.ErrorIs(t, err, errcustom.ErrInsufficientBalance)
}

func TestRecordAdjustment_RequiresNotes(t *testing.T) {
	svc := app.NewStaffFinanceApp(nil, nil, nil)
	_, err := svc.RecordAdjustment(context.Background(), uuid.New(), uuid.New(), financeModels.EntityTypeRider, "credit", 100, "  ")
	require.ErrorIs(t, err, errcustom.ErrFinanceNotesRequired)
}
