package app_test

import (
	"context"
	"testing"
	"time"

	mockuploadimage "Encargalo.app-api.go/internal/mocks/pkg/uploadImage"
	mocksriders "Encargalo.app-api.go/internal/mocks/riders/domain/ports"
	"Encargalo.app-api.go/internal/riders/app"
	"Encargalo.app-api.go/internal/riders/domain/dtos"
	"Encargalo.app-api.go/internal/riders/domain/models"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGetOwnApplication_RiderIsRejected(t *testing.T) {
	repo := mocksriders.NewRiderProfileRepo(t)
	svc := app.NewRiderProfileApp(repo, mockuploadimage.NewUploadImage(t), nil)
	userID := uuid.New()

	repo.On("HasRiderRole", mock.Anything, userID).Return(true, nil)

	_, err := svc.GetOwnApplication(context.Background(), userID)

	assert.ErrorIs(t, err, errcustom.ErrAlreadyRider)
}

func TestGetOwnApplication_RoleLookupFails(t *testing.T) {
	repo := mocksriders.NewRiderProfileRepo(t)
	svc := app.NewRiderProfileApp(repo, mockuploadimage.NewUploadImage(t), nil)
	userID := uuid.New()

	repo.On("HasRiderRole", mock.Anything, userID).Return(false, errcustom.ErrUnexpectedError)

	_, err := svc.GetOwnApplication(context.Background(), userID)

	assert.ErrorIs(t, err, errcustom.ErrUnexpectedError)
}

func TestGetOwnApplication_NoProfileReturnsEditableDraft(t *testing.T) {
	repo := mocksriders.NewRiderProfileRepo(t)
	svc := app.NewRiderProfileApp(repo, mockuploadimage.NewUploadImage(t), nil)
	userID := uuid.New()

	repo.On("HasRiderRole", mock.Anything, userID).Return(false, nil)
	repo.On("GetByUserID", mock.Anything, userID).Return(nil, errcustom.ErrNotFound)

	resp, err := svc.GetOwnApplication(context.Background(), userID)

	require.NoError(t, err)
	assert.Equal(t, string(models.RiderProfileDraft), resp.Status)
	assert.True(t, resp.CanEdit)
	assert.False(t, resp.Completion.IsComplete)
	assert.Contains(t, resp.Completion.Missing, "vehicle_type")
}

func TestGetOwnApplication_PendingReviewIsNotEditable(t *testing.T) {
	repo := mocksriders.NewRiderProfileRepo(t)
	svc := app.NewRiderProfileApp(repo, mockuploadimage.NewUploadImage(t), nil)
	userID := uuid.New()
	submittedAt := time.Date(2026, 10, 20, 15, 0, 0, 0, time.UTC)

	profile := completeProfile(userID, "moto")
	profile.Status = models.RiderProfilePendingReview
	profile.SubmittedAt = &submittedAt
	profile.Notes = new("nota interna del staff")

	repo.On("HasRiderRole", mock.Anything, userID).Return(false, nil)
	repo.On("GetByUserID", mock.Anything, userID).Return(profile, nil)

	resp, err := svc.GetOwnApplication(context.Background(), userID)

	require.NoError(t, err)
	assert.Equal(t, string(models.RiderProfilePendingReview), resp.Status)
	assert.False(t, resp.CanEdit)
	require.NotNil(t, resp.SubmittedAt)
	assert.Equal(t, "2026-10-20T15:00:00Z", *resp.SubmittedAt)
}

func TestSaveOwnApplication_NewApplicationIsSavedAsDraft(t *testing.T) {
	repo := mocksriders.NewRiderProfileRepo(t)
	up := mockuploadimage.NewUploadImage(t)
	stream := mocksriders.NewRidersStream(t)
	svc := app.NewRiderProfileApp(repo, up, stream)
	userID := uuid.New()

	repo.On("HasRiderRole", mock.Anything, userID).Return(false, nil)
	repo.On("GetByUserID", mock.Anything, userID).Return(nil, errcustom.ErrNotFound).Once()
	up.On("UploadImage", mock.Anything, "data:image/png;base64,AAA").
		Return(&uploader.UploadResult{SecureURL: "https://cdn/license.jpg"}, nil)
	repo.On("SaveApplication", mock.Anything, mock.MatchedBy(func(p *models.RiderProfile) bool {
		return p.UserID == userID &&
			p.Status == models.RiderProfileDraft &&
			p.VehicleType != nil && *p.VehicleType == "moto" &&
			p.LicensePhotoURL != nil && *p.LicensePhotoURL == "https://cdn/license.jpg" &&
			p.Notes == nil
	})).Return(nil)
	// El ETL resincroniza DimRider por evento, no por el batch horario.
	stream.On("PublisherEvent", mock.Anything, models.DataEventRiderUpdated{
		MessageType: "rider_updated",
		RiderID:     userID.String(),
	}).Return(nil)

	resp, err := svc.SaveOwnApplication(context.Background(), userID, dtos.RiderApplicationRequest{
		VehicleType:  new(" moto "),
		LicensePhoto: new("data:image/png;base64,AAA"),
	})

	require.NoError(t, err)
	assert.Equal(t, string(models.RiderProfileDraft), resp.Status)
	assert.True(t, resp.CanEdit)
	assert.Equal(t, "https://cdn/license.jpg", *resp.LicensePhotoURL)
}

func TestSaveOwnApplication_ChangesRequestedKeepsStatus(t *testing.T) {
	repo := mocksriders.NewRiderProfileRepo(t)
	svc := app.NewRiderProfileApp(repo, mockuploadimage.NewUploadImage(t), nil)
	userID := uuid.New()

	profile := &models.RiderProfile{UserID: userID, Status: models.RiderProfileChangesRequested}
	repo.On("HasRiderRole", mock.Anything, userID).Return(false, nil)
	repo.On("GetByUserID", mock.Anything, userID).Return(profile, nil)
	repo.On("SaveApplication", mock.Anything, mock.MatchedBy(func(p *models.RiderProfile) bool {
		return p.Status == models.RiderProfileChangesRequested && *p.VehiclePlate == "AB123CD"
	})).Return(nil)

	resp, err := svc.SaveOwnApplication(context.Background(), userID, dtos.RiderApplicationRequest{
		VehiclePlate: new("AB123CD"),
	})

	require.NoError(t, err)
	assert.Equal(t, string(models.RiderProfileChangesRequested), resp.Status)
}

func TestSaveOwnApplication_LockedStatusesDoNotUpload(t *testing.T) {
	statuses := []models.RiderProfileStatus{
		models.RiderProfilePendingReview,
		models.RiderProfileRejected,
		models.RiderProfileActive,
		models.RiderProfileSuspended,
	}
	for _, status := range statuses {
		t.Run(string(status), func(t *testing.T) {
			repo := mocksriders.NewRiderProfileRepo(t)
			svc := app.NewRiderProfileApp(repo, mockuploadimage.NewUploadImage(t), nil)
			userID := uuid.New()

			repo.On("HasRiderRole", mock.Anything, userID).Return(false, nil)
			repo.On("GetByUserID", mock.Anything, userID).Return(&models.RiderProfile{UserID: userID, Status: status}, nil)

			_, err := svc.SaveOwnApplication(context.Background(), userID, dtos.RiderApplicationRequest{
				IDPhoto: new("data:image/png;base64,AAA"),
			})

			assert.ErrorIs(t, err, errcustom.ErrRiderApplicationLocked)
		})
	}
}

func TestSaveOwnApplication_RiderIsRejected(t *testing.T) {
	repo := mocksriders.NewRiderProfileRepo(t)
	svc := app.NewRiderProfileApp(repo, mockuploadimage.NewUploadImage(t), nil)
	userID := uuid.New()

	repo.On("HasRiderRole", mock.Anything, userID).Return(true, nil)

	_, err := svc.SaveOwnApplication(context.Background(), userID, dtos.RiderApplicationRequest{VehicleType: new("moto")})

	assert.ErrorIs(t, err, errcustom.ErrAlreadyRider)
}

func TestSaveOwnApplication_StatusChangedDuringSave(t *testing.T) {
	repo := mocksriders.NewRiderProfileRepo(t)
	svc := app.NewRiderProfileApp(repo, mockuploadimage.NewUploadImage(t), nil)
	userID := uuid.New()

	repo.On("HasRiderRole", mock.Anything, userID).Return(false, nil)
	repo.On("GetByUserID", mock.Anything, userID).Return(&models.RiderProfile{UserID: userID, Status: models.RiderProfileDraft}, nil)
	repo.On("SaveApplication", mock.Anything, mock.Anything).Return(errcustom.ErrRiderApplicationLocked)

	_, err := svc.SaveOwnApplication(context.Background(), userID, dtos.RiderApplicationRequest{VehicleType: new("moto")})

	assert.ErrorIs(t, err, errcustom.ErrRiderApplicationLocked)
}

func TestSaveOwnApplication_ReplacedPhotoIsDeletedAfterSave(t *testing.T) {
	repo := mocksriders.NewRiderProfileRepo(t)
	up := mockuploadimage.NewUploadImage(t)
	svc := app.NewRiderProfileApp(repo, up, nil)
	userID := uuid.New()

	repo.On("HasRiderRole", mock.Anything, userID).Return(false, nil)
	repo.On("GetByUserID", mock.Anything, userID).Return(&models.RiderProfile{
		UserID:     userID,
		Status:     models.RiderProfileDraft,
		IDPhotoURL: new("https://cdn/old-id.jpg"),
	}, nil)
	up.On("UploadImage", mock.Anything, "data:image/png;base64,NEW").
		Return(&uploader.UploadResult{SecureURL: "https://cdn/new-id.jpg"}, nil)
	repo.On("SaveApplication", mock.Anything, mock.Anything).Return(nil)
	up.On("DeleteImage", mock.Anything, "https://cdn/old-id.jpg").Return(nil).Once()

	resp, err := svc.SaveOwnApplication(context.Background(), userID, dtos.RiderApplicationRequest{
		IDPhoto: new("data:image/png;base64,NEW"),
	})

	require.NoError(t, err)
	assert.Equal(t, "https://cdn/new-id.jpg", *resp.IDPhotoURL)
	repo.AssertNumberOfCalls(t, "GetByUserID", 1)
}

func TestSaveOwnApplication_LockedDuringSaveDeletesUploadedPhotos(t *testing.T) {
	repo := mocksriders.NewRiderProfileRepo(t)
	up := mockuploadimage.NewUploadImage(t)
	svc := app.NewRiderProfileApp(repo, up, nil)
	userID := uuid.New()

	repo.On("HasRiderRole", mock.Anything, userID).Return(false, nil)
	repo.On("GetByUserID", mock.Anything, userID).Return(&models.RiderProfile{
		UserID:     userID,
		Status:     models.RiderProfileDraft,
		IDPhotoURL: new("https://cdn/old-id.jpg"),
	}, nil)
	up.On("UploadImage", mock.Anything, "data:image/png;base64,NEW").
		Return(&uploader.UploadResult{SecureURL: "https://cdn/new-id.jpg"}, nil)
	repo.On("SaveApplication", mock.Anything, mock.Anything).Return(errcustom.ErrRiderApplicationLocked)
	up.On("DeleteImage", mock.Anything, "https://cdn/new-id.jpg").Return(nil).Once()

	_, err := svc.SaveOwnApplication(context.Background(), userID, dtos.RiderApplicationRequest{
		IDPhoto: new("data:image/png;base64,NEW"),
	})

	assert.ErrorIs(t, err, errcustom.ErrRiderApplicationLocked)
	up.AssertNotCalled(t, "DeleteImage", mock.Anything, "https://cdn/old-id.jpg")
}

func TestSaveOwnApplication_PartialUploadFailureDeletesUploadedPhotos(t *testing.T) {
	repo := mocksriders.NewRiderProfileRepo(t)
	up := mockuploadimage.NewUploadImage(t)
	svc := app.NewRiderProfileApp(repo, up, nil)
	userID := uuid.New()

	repo.On("HasRiderRole", mock.Anything, userID).Return(false, nil)
	repo.On("GetByUserID", mock.Anything, userID).Return(nil, errcustom.ErrNotFound)
	up.On("UploadImage", mock.Anything, "data:image/png;base64,OK").
		Return(&uploader.UploadResult{SecureURL: "https://cdn/ok.jpg"}, nil)
	up.On("UploadImage", mock.Anything, "data:image/png;base64,BAD").Return(nil, errcustom.ErrUploadingImage)
	up.On("DeleteImage", mock.Anything, "https://cdn/ok.jpg").Return(nil).Once()

	_, err := svc.SaveOwnApplication(context.Background(), userID, dtos.RiderApplicationRequest{
		IDPhoto:      new("data:image/png;base64,OK"),
		LicensePhoto: new("data:image/png;base64,BAD"),
	})

	assert.ErrorIs(t, err, errcustom.ErrUploadingImage)
	repo.AssertNotCalled(t, "SaveApplication", mock.Anything, mock.Anything)
}

func TestSaveOwnApplication_UploadFails(t *testing.T) {
	repo := mocksriders.NewRiderProfileRepo(t)
	up := mockuploadimage.NewUploadImage(t)
	svc := app.NewRiderProfileApp(repo, up, nil)
	userID := uuid.New()

	repo.On("HasRiderRole", mock.Anything, userID).Return(false, nil)
	repo.On("GetByUserID", mock.Anything, userID).Return(nil, errcustom.ErrNotFound)
	up.On("UploadImage", mock.Anything, mock.Anything).Return(nil, errcustom.ErrUploadingImage)

	_, err := svc.SaveOwnApplication(context.Background(), userID, dtos.RiderApplicationRequest{
		IDPhoto: new("data:image/png;base64,AAA"),
	})

	assert.ErrorIs(t, err, errcustom.ErrUploadingImage)
}

func TestSubmitOwnApplication_CompleteDraftGoesToReview(t *testing.T) {
	repo := mocksriders.NewRiderProfileRepo(t)
	stream := mocksriders.NewRidersStream(t)
	svc := app.NewRiderProfileApp(repo, mockuploadimage.NewUploadImage(t), stream)
	userID := uuid.New()

	repo.On("HasRiderRole", mock.Anything, userID).Return(false, nil)
	repo.On("GetByUserID", mock.Anything, userID).Return(completeProfile(userID, "moto"), nil).Once()
	repo.On("SubmitApplication", mock.Anything, userID, mock.AnythingOfType("time.Time")).Return(nil)
	stream.On("PublisherEvent", mock.Anything, models.DataEventRiderUpdated{
		MessageType: "rider_updated",
		RiderID:     userID.String(),
	}).Return(nil)

	resp, err := svc.SubmitOwnApplication(context.Background(), userID)

	require.NoError(t, err)
	assert.Equal(t, string(models.RiderProfilePendingReview), resp.Status)
	assert.False(t, resp.CanEdit)
	assert.NotNil(t, resp.SubmittedAt)
	repo.AssertNumberOfCalls(t, "GetByUserID", 1)
	repo.AssertNumberOfCalls(t, "HasRiderRole", 1)
	repo.AssertNotCalled(t, "EnsureRiderRole", mock.Anything, mock.Anything)
	repo.AssertNotCalled(t, "Activate", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestSubmitOwnApplication_ChangesRequestedCanResubmit(t *testing.T) {
	repo := mocksriders.NewRiderProfileRepo(t)
	svc := app.NewRiderProfileApp(repo, mockuploadimage.NewUploadImage(t), nil)
	userID := uuid.New()

	profile := completeProfile(userID, "bici")
	profile.VehiclePlate = nil
	profile.Status = models.RiderProfileChangesRequested

	repo.On("HasRiderRole", mock.Anything, userID).Return(false, nil)
	repo.On("GetByUserID", mock.Anything, userID).Return(profile, nil)
	repo.On("SubmitApplication", mock.Anything, userID, mock.AnythingOfType("time.Time")).Return(nil)

	_, err := svc.SubmitOwnApplication(context.Background(), userID)

	require.NoError(t, err)
}

func TestSubmitOwnApplication_IncompleteIsRejected(t *testing.T) {
	repo := mocksriders.NewRiderProfileRepo(t)
	svc := app.NewRiderProfileApp(repo, mockuploadimage.NewUploadImage(t), nil)
	userID := uuid.New()

	repo.On("HasRiderRole", mock.Anything, userID).Return(false, nil)
	repo.On("GetByUserID", mock.Anything, userID).Return(&models.RiderProfile{
		UserID:      userID,
		Status:      models.RiderProfileDraft,
		VehicleType: new("moto"),
	}, nil)

	resp, err := svc.SubmitOwnApplication(context.Background(), userID)

	assert.ErrorIs(t, err, errcustom.ErrRiderProfileIncomplete)
	assert.Contains(t, resp.Completion.Missing, "id_photo")
	assert.NotContains(t, resp.Completion.Missing, "vehicle_type")
	repo.AssertNotCalled(t, "SubmitApplication", mock.Anything, mock.Anything, mock.Anything)
}

func TestSubmitOwnApplication_WithoutApplicationIsIncomplete(t *testing.T) {
	repo := mocksriders.NewRiderProfileRepo(t)
	svc := app.NewRiderProfileApp(repo, mockuploadimage.NewUploadImage(t), nil)
	userID := uuid.New()

	repo.On("HasRiderRole", mock.Anything, userID).Return(false, nil)
	repo.On("GetByUserID", mock.Anything, userID).Return(nil, errcustom.ErrNotFound)

	_, err := svc.SubmitOwnApplication(context.Background(), userID)

	assert.ErrorIs(t, err, errcustom.ErrRiderProfileIncomplete)
}

func TestSubmitOwnApplication_AlreadySubmittedIsLocked(t *testing.T) {
	repo := mocksriders.NewRiderProfileRepo(t)
	svc := app.NewRiderProfileApp(repo, mockuploadimage.NewUploadImage(t), nil)
	userID := uuid.New()

	profile := completeProfile(userID, "moto")
	profile.Status = models.RiderProfilePendingReview

	repo.On("HasRiderRole", mock.Anything, userID).Return(false, nil)
	repo.On("GetByUserID", mock.Anything, userID).Return(profile, nil)

	_, err := svc.SubmitOwnApplication(context.Background(), userID)

	assert.ErrorIs(t, err, errcustom.ErrRiderApplicationLocked)
}

func TestSubmitOwnApplication_RiderIsRejected(t *testing.T) {
	repo := mocksriders.NewRiderProfileRepo(t)
	svc := app.NewRiderProfileApp(repo, mockuploadimage.NewUploadImage(t), nil)
	userID := uuid.New()

	repo.On("HasRiderRole", mock.Anything, userID).Return(true, nil)

	_, err := svc.SubmitOwnApplication(context.Background(), userID)

	assert.ErrorIs(t, err, errcustom.ErrAlreadyRider)
}
