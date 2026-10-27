package app_test

import (
	"context"
	"testing"

	mockuploadimage "Encargalo.app-api.go/internal/mocks/pkg/uploadImage"
	mocksriders "Encargalo.app-api.go/internal/mocks/riders/domain/ports"
	"Encargalo.app-api.go/internal/riders/app"
	"Encargalo.app-api.go/internal/riders/domain/models"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// pendingProfile es una solicitud completa ya enviada, que es lo único que el staff puede decidir.
func pendingProfile(userID uuid.UUID) *models.RiderProfile {
	profile := completeProfile(userID, "moto")
	profile.Status = models.RiderProfilePendingReview
	return profile
}

func TestRequestApplicationChanges_LeavesChangesRequested(t *testing.T) {
	repo := mocksriders.NewRiderProfileRepo(t)
	stream := mocksriders.NewRidersStream(t)
	svc := app.NewRiderProfileApp(repo, mockuploadimage.NewUploadImage(t), stream)
	userID, staffID := uuid.New(), uuid.New()
	reason := "La foto de la licencia está borrosa"

	reviewed := pendingProfile(userID)
	reviewed.Status = models.RiderProfileChangesRequested
	reviewed.ReviewReason = &reason

	repo.On("GetByUserID", mock.Anything, userID).Return(pendingProfile(userID), nil).Once()
	repo.On("ReviewApplication", mock.Anything, userID, staffID,
		models.RiderProfileChangesRequested, reason, mock.Anything).Return(nil)
	repo.On("GetByUserID", mock.Anything, userID).Return(reviewed, nil).Once()
	repo.On("GetAccountBasics", mock.Anything, userID).Return("Rider", "Prueba", "+584120000284", "r@x.com", nil)
	repo.On("HasRiderRole", mock.Anything, userID).Return(false, nil)
	// El ETL resincroniza DimRider por evento, no por el batch horario; cubre
	// también RejectApplication, que comparte el helper review().
	stream.On("PublisherEvent", mock.Anything, models.DataEventRiderUpdated{
		MessageType: "rider_updated",
		RiderID:     userID.String(),
	}).Return(nil)

	resp, err := svc.RequestApplicationChanges(context.Background(), userID, staffID, reason)

	require.NoError(t, err)
	assert.Equal(t, string(models.RiderProfileChangesRequested), resp.Status)
	require.NotNil(t, resp.ReviewReason)
	assert.Equal(t, reason, *resp.ReviewReason)
	assert.False(t, resp.HasRiderRole)
}

func TestRejectApplication_LeavesRejected(t *testing.T) {
	repo := mocksriders.NewRiderProfileRepo(t)
	svc := app.NewRiderProfileApp(repo, mockuploadimage.NewUploadImage(t), nil)
	userID, staffID := uuid.New(), uuid.New()
	reason := "Los documentos no corresponden al titular"

	reviewed := pendingProfile(userID)
	reviewed.Status = models.RiderProfileRejected
	reviewed.ReviewReason = &reason

	repo.On("GetByUserID", mock.Anything, userID).Return(pendingProfile(userID), nil).Once()
	repo.On("ReviewApplication", mock.Anything, userID, staffID,
		models.RiderProfileRejected, reason, mock.Anything).Return(nil)
	repo.On("GetByUserID", mock.Anything, userID).Return(reviewed, nil).Once()
	repo.On("GetAccountBasics", mock.Anything, userID).Return("Rider", "Prueba", "+584120000284", "r@x.com", nil)
	repo.On("HasRiderRole", mock.Anything, userID).Return(false, nil)

	resp, err := svc.RejectApplication(context.Background(), userID, staffID, reason)

	require.NoError(t, err)
	assert.Equal(t, string(models.RiderProfileRejected), resp.Status)
}

func TestReview_TrimsReasonBeforeSaving(t *testing.T) {
	repo := mocksriders.NewRiderProfileRepo(t)
	svc := app.NewRiderProfileApp(repo, mockuploadimage.NewUploadImage(t), nil)
	userID, staffID := uuid.New(), uuid.New()

	repo.On("GetByUserID", mock.Anything, userID).Return(pendingProfile(userID), nil).Once()
	repo.On("ReviewApplication", mock.Anything, userID, staffID,
		models.RiderProfileChangesRequested, "Falta el carnet", mock.Anything).Return(nil)
	repo.On("GetByUserID", mock.Anything, userID).Return(pendingProfile(userID), nil).Once()
	repo.On("GetAccountBasics", mock.Anything, userID).Return("Rider", "Prueba", "", "", nil)
	repo.On("HasRiderRole", mock.Anything, userID).Return(false, nil)

	_, err := svc.RequestApplicationChanges(context.Background(), userID, staffID, "  Falta el carnet  ")

	require.NoError(t, err)
}

func TestReview_EmptyReasonIsRejectedBeforeTouchingTheProfile(t *testing.T) {
	repo := mocksriders.NewRiderProfileRepo(t)
	svc := app.NewRiderProfileApp(repo, mockuploadimage.NewUploadImage(t), nil)

	_, err := svc.RequestApplicationChanges(context.Background(), uuid.New(), uuid.New(), "   ")

	assert.ErrorIs(t, err, errcustom.ErrReviewReasonRequired)
}

func TestReview_OnlyPendingReviewCanBeDecided(t *testing.T) {
	for _, status := range []models.RiderProfileStatus{
		models.RiderProfileDraft,
		models.RiderProfileChangesRequested,
		models.RiderProfileRejected,
		models.RiderProfileActive,
		models.RiderProfileSuspended,
	} {
		t.Run(string(status), func(t *testing.T) {
			repo := mocksriders.NewRiderProfileRepo(t)
			svc := app.NewRiderProfileApp(repo, mockuploadimage.NewUploadImage(t), nil)
			userID := uuid.New()

			profile := pendingProfile(userID)
			profile.Status = status
			repo.On("GetByUserID", mock.Anything, userID).Return(profile, nil)

			_, err := svc.RejectApplication(context.Background(), userID, uuid.New(), "Un motivo cualquiera")

			assert.ErrorIs(t, err, errcustom.ErrRiderReviewNotPending)
		})
	}
}

func TestReview_MissingProfileIsNotFound(t *testing.T) {
	repo := mocksriders.NewRiderProfileRepo(t)
	svc := app.NewRiderProfileApp(repo, mockuploadimage.NewUploadImage(t), nil)
	userID := uuid.New()

	repo.On("GetByUserID", mock.Anything, userID).Return(nil, errcustom.ErrNotFound)

	_, err := svc.RequestApplicationChanges(context.Background(), userID, uuid.New(), "Un motivo cualquiera")

	assert.ErrorIs(t, err, errcustom.ErrNotFound)
}

// El repo condiciona el UPDATE a pending_review: si alguien decidió en paralelo, el error sube tal cual.
func TestReview_ConcurrentDecisionBubblesUp(t *testing.T) {
	repo := mocksriders.NewRiderProfileRepo(t)
	svc := app.NewRiderProfileApp(repo, mockuploadimage.NewUploadImage(t), nil)
	userID, staffID := uuid.New(), uuid.New()

	repo.On("GetByUserID", mock.Anything, userID).Return(pendingProfile(userID), nil)
	repo.On("ReviewApplication", mock.Anything, userID, staffID,
		models.RiderProfileRejected, "Un motivo cualquiera", mock.Anything).
		Return(errcustom.ErrRiderReviewNotPending)

	_, err := svc.RejectApplication(context.Background(), userID, staffID, "Un motivo cualquiera")

	assert.ErrorIs(t, err, errcustom.ErrRiderReviewNotPending)
}
