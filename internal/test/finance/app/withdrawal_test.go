package app_test

import (
	"context"
	"testing"
	"time"

	bcvmodels "Encargalo.app-api.go/internal/bcv/domain/models"
	"Encargalo.app-api.go/internal/finance/app"
	financedtos "Encargalo.app-api.go/internal/finance/domain/dtos"
	financeModels "Encargalo.app-api.go/internal/finance/domain/models"
	mockbcv "Encargalo.app-api.go/internal/mocks/bcv/domain/ports"
	mockfinance "Encargalo.app-api.go/internal/mocks/finance/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// El listado tiene que exponer lo solicitado y lo pagable por separado: si el
// CRM muestra un solo número, el operador transfiere de más.
func TestListWithdrawals_ExposesRequestedAndPayableSeparately(t *testing.T) {
	repo := mockfinance.NewWithdrawalRepo(t)
	bcv := mockbcv.NewBcvApp(t)
	svc := app.NewWithdrawalApp(repo, bcv, nil)

	sur := "Pérez"
	phone := "04121234567"
	repo.On("List", mock.Anything, financeModels.EntityTypeRider, mock.Anything).Return([]financeModels.WithdrawalListRow{
		{
			WithdrawalRequest: financeModels.WithdrawalRequest{
				ID:        uuid.New(),
				EntityID:  uuid.New(),
				Amount:    800,
				Currency:  financeModels.CurrencyVES,
				Status:    financeModels.WithdrawalPending,
				PMPhone:   &phone,
				CreatedAt: time.Now(),
			},
			Name:             "Ana",
			SurName:          &sur,
			AvailableBalance: -728,
		},
	}, nil)
	bcv.On("GetRate", mock.Anything).Return(&bcvmodels.ExchangeRate{Rate: 40}, nil)

	resp, err := svc.List(context.Background(), financeModels.EntityTypeRider, nil)

	require.NoError(t, err)
	require.Equal(t, 1, resp.Total)
	assert.Equal(t, 800.0, resp.Items[0].AmountBs)
	assert.Equal(t, 72.0, resp.Items[0].PayableBs)
	assert.False(t, resp.Items[0].PayableFrozen)
	assert.Equal(t, "Pérez", resp.Items[0].SurName)
}

func TestListWithdrawals_MarksFrozenPayableAfterClaim(t *testing.T) {
	repo := mockfinance.NewWithdrawalRepo(t)
	bcv := mockbcv.NewBcvApp(t)
	svc := app.NewWithdrawalApp(repo, bcv, nil)

	frozen := 72.0
	repo.On("List", mock.Anything, financeModels.EntityTypeRider, mock.Anything).Return([]financeModels.WithdrawalListRow{
		{
			WithdrawalRequest: financeModels.WithdrawalRequest{
				ID: uuid.New(), EntityID: uuid.New(), Amount: 800,
				Currency: financeModels.CurrencyVES, Status: financeModels.WithdrawalProcessing,
				PayableAmount: &frozen, CreatedAt: time.Now(),
			},
			Name:             "Ana",
			AvailableBalance: -728,
		},
	}, nil)
	bcv.On("GetRate", mock.Anything).Return(&bcvmodels.ExchangeRate{Rate: 40}, nil)

	resp, err := svc.List(context.Background(), financeModels.EntityTypeRider, nil)

	require.NoError(t, err)
	assert.Equal(t, 72.0, resp.Items[0].PayableBs)
	assert.True(t, resp.Items[0].PayableFrozen)
}

func TestClaimWithdrawal_ReturnsFrozenAmountToTransfer(t *testing.T) {
	repo := mockfinance.NewWithdrawalRepo(t)
	bcv := mockbcv.NewBcvApp(t)
	svc := app.NewWithdrawalApp(repo, bcv, nil)

	requestID := uuid.New()
	staffID := uuid.New()
	frozen := 72.0
	now := time.Now()
	phone := "04121234567"
	repo.On("Claim", mock.Anything, requestID, staffID).Return(financeModels.WithdrawalRequest{
		ID: requestID, EntityType: financeModels.EntityTypeRider, Amount: 800,
		PayableAmount: &frozen, Currency: financeModels.CurrencyVES,
		Status: financeModels.WithdrawalProcessing, ProcessingAt: &now, PMPhone: &phone,
	}, nil)
	bcv.On("GetRate", mock.Anything).Return(&bcvmodels.ExchangeRate{Rate: 40}, nil)

	resp, err := svc.Claim(context.Background(), requestID, staffID)

	require.NoError(t, err)
	assert.Equal(t, 72.0, resp.PayableBs)
	assert.Equal(t, "04121234567", resp.PMPhone)
	assert.Equal(t, string(financeModels.WithdrawalProcessing), resp.Status)
}

func TestClaimWithdrawal_NothingPayable(t *testing.T) {
	repo := mockfinance.NewWithdrawalRepo(t)
	svc := app.NewWithdrawalApp(repo, nil, nil)

	requestID := uuid.New()
	staffID := uuid.New()
	repo.On("Claim", mock.Anything, requestID, staffID).
		Return(financeModels.WithdrawalRequest{}, errcustom.ErrWithdrawalNothingPayable)

	_, err := svc.Claim(context.Background(), requestID, staffID)

	assert.ErrorIs(t, err, errcustom.ErrWithdrawalNothingPayable)
}

// Sin comprobante no se cierra, y se falla antes de tocar la BD.
func TestCompleteWithdrawal_ProofRequired(t *testing.T) {
	repo := mockfinance.NewWithdrawalRepo(t)
	svc := app.NewWithdrawalApp(repo, nil, nil)

	_, err := svc.Complete(context.Background(), uuid.New(), uuid.New(), financedtos.CompleteWithdrawalRequest{ProofImage: "   "})

	assert.ErrorIs(t, err, errcustom.ErrWithdrawalProofRequired)
}

// El estado se valida antes de subir la imagen: no tiene sentido dejar un
// comprobante huérfano en Cloudinary por una solicitud que no se puede pagar.
func TestCompleteWithdrawal_RejectsUnclaimedBeforeUpload(t *testing.T) {
	repo := mockfinance.NewWithdrawalRepo(t)
	uploader := &failingUploader{}
	svc := app.NewWithdrawalApp(repo, nil, uploader)

	requestID := uuid.New()
	repo.On("GetByID", mock.Anything, requestID).Return(financeModels.WithdrawalRequest{
		ID: requestID, Status: financeModels.WithdrawalPending,
	}, nil)

	_, err := svc.Complete(context.Background(), requestID, uuid.New(), financedtos.CompleteWithdrawalRequest{ProofImage: "data:image/png;base64,abc"})

	assert.ErrorIs(t, err, errcustom.ErrWithdrawalInvalidTransition)
	assert.False(t, uploader.called, "no debe subir el comprobante si la solicitud no admite el pago")
}

func TestCompleteWithdrawal_UploadFailureLeavesRequestUntouched(t *testing.T) {
	repo := mockfinance.NewWithdrawalRepo(t)
	svc := app.NewWithdrawalApp(repo, nil, &failingUploader{})

	requestID := uuid.New()
	frozen := 72.0
	repo.On("GetByID", mock.Anything, requestID).Return(financeModels.WithdrawalRequest{
		ID: requestID, Status: financeModels.WithdrawalProcessing, PayableAmount: &frozen,
	}, nil)

	_, err := svc.Complete(context.Background(), requestID, uuid.New(), financedtos.CompleteWithdrawalRequest{ProofImage: "data:image/png;base64,abc"})

	assert.ErrorIs(t, err, errcustom.ErrUploadingImage)
	// Complete nunca se llamó: la solicitud sigue reclamada, sin pago registrado.
	repo.AssertNotCalled(t, "Complete", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestRejectWithdrawal_ReasonRequired(t *testing.T) {
	repo := mockfinance.NewWithdrawalRepo(t)
	svc := app.NewWithdrawalApp(repo, nil, nil)

	_, err := svc.Reject(context.Background(), uuid.New(), uuid.New(), financedtos.RejectWithdrawalRequest{Reason: "  "})

	assert.ErrorIs(t, err, errcustom.ErrWithdrawalReasonRequired)
}

func TestRejectWithdrawal_Success(t *testing.T) {
	repo := mockfinance.NewWithdrawalRepo(t)
	bcv := mockbcv.NewBcvApp(t)
	svc := app.NewWithdrawalApp(repo, bcv, nil)

	requestID := uuid.New()
	staffID := uuid.New()
	reason := "el Pago Móvil no corresponde al rider"
	repo.On("Reject", mock.Anything, requestID, staffID, reason).Return(financeModels.WithdrawalRequest{
		ID: requestID, EntityType: financeModels.EntityTypeRider, Amount: 800,
		Currency: financeModels.CurrencyVES, Status: financeModels.WithdrawalRejected,
		RejectionReason: &reason,
	}, nil)
	bcv.On("GetRate", mock.Anything).Return(&bcvmodels.ExchangeRate{Rate: 40}, nil)

	resp, err := svc.Reject(context.Background(), requestID, staffID, financedtos.RejectWithdrawalRequest{Reason: reason})

	require.NoError(t, err)
	assert.Equal(t, string(financeModels.WithdrawalRejected), resp.Status)
	require.NotNil(t, resp.RejectionReason)
	assert.Equal(t, reason, *resp.RejectionReason)
}

// failingUploader simula una falla de Cloudinary y registra si se lo llamó.
type failingUploader struct {
	called bool
}

func (f *failingUploader) UploadImage(ctx context.Context, imageB64 string) (*uploader.UploadResult, error) {
	f.called = true
	return nil, errcustom.ErrUploadingImage
}

func (f *failingUploader) DeleteImage(ctx context.Context, secureURL string) error {
	return nil
}
