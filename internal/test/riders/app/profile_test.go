package app_test

import (
	"context"
	"testing"

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

func TestFindCandidate_NoProfileYet(t *testing.T) {
	repo := mocksriders.NewRiderProfileRepo(t)
	svc := app.NewRiderProfileApp(repo, mockuploadimage.NewUploadImage(t), nil)

	userID := uuid.New()
	repo.On("FindAccountByPhone", mock.Anything, "0414").Return(userID, "Ada", "Lovelace", "0414", "a@b.c", nil)
	repo.On("HasRiderRole", mock.Anything, userID).Return(false, nil)
	repo.On("GetByUserID", mock.Anything, userID).Return(nil, errcustom.ErrNotFound)

	resp, err := svc.FindCandidate(context.Background(), "0414")

	require.NoError(t, err)
	assert.Equal(t, userID.String(), resp.UserID)
	assert.Nil(t, resp.Profile)
}

func TestFindCandidate_WithExistingProfile(t *testing.T) {
	repo := mocksriders.NewRiderProfileRepo(t)
	svc := app.NewRiderProfileApp(repo, mockuploadimage.NewUploadImage(t), nil)

	userID := uuid.New()
	repo.On("FindAccountByPhone", mock.Anything, "0414").Return(userID, "Ada", "Lovelace", "0414", "a@b.c", nil)
	repo.On("HasRiderRole", mock.Anything, userID).Return(true, nil)
	repo.On("GetByUserID", mock.Anything, userID).Return(&models.RiderProfile{UserID: userID, Status: models.RiderProfileDraft}, nil)

	resp, err := svc.FindCandidate(context.Background(), "0414")

	require.NoError(t, err)
	require.NotNil(t, resp.Profile)
	assert.True(t, resp.HasRiderRole)
}

func TestFindCandidate_AccountNotFound(t *testing.T) {
	repo := mocksriders.NewRiderProfileRepo(t)
	svc := app.NewRiderProfileApp(repo, mockuploadimage.NewUploadImage(t), nil)

	repo.On("FindAccountByPhone", mock.Anything, "0414").Return(uuid.Nil, "", "", "", "", errcustom.ErrNotFound)

	_, err := svc.FindCandidate(context.Background(), "0414")

	assert.ErrorIs(t, err, errcustom.ErrNotFound)
}

func TestGetProfile_AccountBasicsError(t *testing.T) {
	repo := mocksriders.NewRiderProfileRepo(t)
	svc := app.NewRiderProfileApp(repo, mockuploadimage.NewUploadImage(t), nil)

	userID := uuid.New()
	repo.On("GetByUserID", mock.Anything, userID).Return(&models.RiderProfile{UserID: userID}, nil)
	repo.On("GetAccountBasics", mock.Anything, userID).Return("", "", "", "", errcustom.ErrUnexpectedError)

	_, err := svc.GetProfile(context.Background(), userID)

	assert.ErrorIs(t, err, errcustom.ErrUnexpectedError)
}

func TestListProfiles_DefaultsToPendingReviewAndSkipsBrokenAccounts(t *testing.T) {
	repo := mocksriders.NewRiderProfileRepo(t)
	svc := app.NewRiderProfileApp(repo, mockuploadimage.NewUploadImage(t), nil)

	ok := uuid.New()
	broken := uuid.New()
	repo.On("ListByStatus", mock.Anything, models.RiderProfilePendingReview).Return([]models.RiderProfile{
		{UserID: ok, Status: models.RiderProfilePendingReview},
		{UserID: broken, Status: models.RiderProfilePendingReview},
	}, nil)
	repo.On("GetAccountBasics", mock.Anything, ok).Return("Ada", "Lovelace", "0414", "a@b.c", nil)
	repo.On("HasRiderRole", mock.Anything, ok).Return(true, nil)
	repo.On("GetAccountBasics", mock.Anything, broken).Return("", "", "", "", errcustom.ErrUnexpectedError)

	out, err := svc.ListProfiles(context.Background(), "")

	require.NoError(t, err)
	require.Len(t, out, 1)
	assert.Equal(t, ok.String(), out[0].UserID)
}

func TestListProfiles_RepoError(t *testing.T) {
	repo := mocksriders.NewRiderProfileRepo(t)
	svc := app.NewRiderProfileApp(repo, mockuploadimage.NewUploadImage(t), nil)

	repo.On("ListByStatus", mock.Anything, models.RiderProfileActive).Return(nil, errcustom.ErrUnexpectedError)

	_, err := svc.ListProfiles(context.Background(), models.RiderProfileActive)

	assert.ErrorIs(t, err, errcustom.ErrUnexpectedError)
}

func TestUpsertProfile_CreatesDraftWithoutPhotos(t *testing.T) {
	repo := mocksriders.NewRiderProfileRepo(t)
	stream := mocksriders.NewRidersStream(t)
	svc := app.NewRiderProfileApp(repo, mockuploadimage.NewUploadImage(t), stream)

	userID := uuid.New()
	repo.On("GetAccountBasics", mock.Anything, userID).Return("Ada", "Lovelace", "0414", "a@b.c", nil).Twice()
	repo.On("GetByUserID", mock.Anything, userID).Return(nil, errcustom.ErrNotFound).Once()
	repo.On("Upsert", mock.Anything, mock.MatchedBy(func(p *models.RiderProfile) bool {
		return p.UserID == userID && p.Status == models.RiderProfileDraft && *p.VehicleType == "bici"
	})).Return(nil)
	repo.On("GetByUserID", mock.Anything, userID).Return(&models.RiderProfile{UserID: userID, Status: models.RiderProfileDraft}, nil).Once()
	repo.On("HasRiderRole", mock.Anything, userID).Return(false, nil)
	// El ETL resincroniza DimRider por evento, no por el batch horario.
	stream.On("PublisherEvent", mock.Anything, models.DataEventRiderUpdated{
		MessageType: "rider_updated",
		RiderID:     userID.String(),
	}).Return(nil)

	vehicleType := "bici"
	resp, err := svc.UpsertProfile(context.Background(), userID, dtos.UpsertRiderProfileRequest{VehicleType: &vehicleType})

	require.NoError(t, err)
	assert.Equal(t, userID.String(), resp.UserID)
}

func TestUpsertProfile_AccountNotFound(t *testing.T) {
	repo := mocksriders.NewRiderProfileRepo(t)
	svc := app.NewRiderProfileApp(repo, mockuploadimage.NewUploadImage(t), nil)

	userID := uuid.New()
	repo.On("GetAccountBasics", mock.Anything, userID).Return("", "", "", "", errcustom.ErrNotFound)

	_, err := svc.UpsertProfile(context.Background(), userID, dtos.UpsertRiderProfileRequest{})

	assert.ErrorIs(t, err, errcustom.ErrNotFound)
}

func TestUpsertProfile_UploadsPhotosConcurrently(t *testing.T) {
	repo := mocksriders.NewRiderProfileRepo(t)
	uploader_ := mockuploadimage.NewUploadImage(t)
	svc := app.NewRiderProfileApp(repo, uploader_, nil)

	userID := uuid.New()
	repo.On("GetAccountBasics", mock.Anything, userID).Return("Ada", "Lovelace", "0414", "a@b.c", nil).Twice()
	repo.On("GetByUserID", mock.Anything, userID).Return(nil, errcustom.ErrNotFound).Once()
	uploader_.On("UploadImage", mock.Anything, "base64photo").Return(&uploader.UploadResult{SecureURL: "https://cdn/x.jpg"}, nil)
	repo.On("Upsert", mock.Anything, mock.MatchedBy(func(p *models.RiderProfile) bool {
		return p.ProfilePhotoURL != nil && *p.ProfilePhotoURL == "https://cdn/x.jpg"
	})).Return(nil)
	repo.On("GetByUserID", mock.Anything, userID).Return(&models.RiderProfile{UserID: userID, Status: models.RiderProfileDraft}, nil).Once()
	repo.On("HasRiderRole", mock.Anything, userID).Return(false, nil)

	photo := "base64photo"
	_, err := svc.UpsertProfile(context.Background(), userID, dtos.UpsertRiderProfileRequest{ProfilePhoto: &photo})

	require.NoError(t, err)
}

func TestUpsertProfile_ReplacedPhotoIsDeletedAfterSave(t *testing.T) {
	repo := mocksriders.NewRiderProfileRepo(t)
	uploader_ := mockuploadimage.NewUploadImage(t)
	svc := app.NewRiderProfileApp(repo, uploader_, nil)

	userID := uuid.New()
	repo.On("GetAccountBasics", mock.Anything, userID).Return("Ada", "Lovelace", "0414", "a@b.c", nil).Twice()
	repo.On("GetByUserID", mock.Anything, userID).Return(&models.RiderProfile{
		UserID:          userID,
		Status:          models.RiderProfileActive,
		ProfilePhotoURL: new("https://cdn/old.jpg"),
	}, nil).Once()
	uploader_.On("UploadImage", mock.Anything, "base64photo").Return(&uploader.UploadResult{SecureURL: "https://cdn/new.jpg"}, nil)
	repo.On("Upsert", mock.Anything, mock.Anything).Return(nil)
	uploader_.On("DeleteImage", mock.Anything, "https://cdn/old.jpg").Return(nil).Once()
	repo.On("GetByUserID", mock.Anything, userID).Return(&models.RiderProfile{UserID: userID, Status: models.RiderProfileActive}, nil).Once()
	repo.On("HasRiderRole", mock.Anything, userID).Return(true, nil)

	_, err := svc.UpsertProfile(context.Background(), userID, dtos.UpsertRiderProfileRequest{ProfilePhoto: new("base64photo")})

	require.NoError(t, err)
}

func TestUpsertProfile_UploadFails(t *testing.T) {
	repo := mocksriders.NewRiderProfileRepo(t)
	uploader_ := mockuploadimage.NewUploadImage(t)
	svc := app.NewRiderProfileApp(repo, uploader_, nil)

	userID := uuid.New()
	repo.On("GetAccountBasics", mock.Anything, userID).Return("Ada", "Lovelace", "0414", "a@b.c", nil).Once()
	repo.On("GetByUserID", mock.Anything, userID).Return(nil, errcustom.ErrNotFound).Once()
	uploader_.On("UploadImage", mock.Anything, "bad").Return(nil, errcustom.ErrUploadingImage)

	photo := "bad"
	_, err := svc.UpsertProfile(context.Background(), userID, dtos.UpsertRiderProfileRequest{ProfilePhoto: &photo})

	assert.ErrorIs(t, err, errcustom.ErrUploadingImage)
}

func TestActivate_Success(t *testing.T) {
	repo := mocksriders.NewRiderProfileRepo(t)
	stream := mocksriders.NewRidersStream(t)
	svc := app.NewRiderProfileApp(repo, mockuploadimage.NewUploadImage(t), stream)

	userID, staffID := uuid.New(), uuid.New()
	vt := "moto"
	complete := completeProfile(userID, vt)
	repo.On("GetByUserID", mock.Anything, userID).Return(complete, nil).Once()
	repo.On("EnsureRiderRole", mock.Anything, userID).Return(nil)
	repo.On("Activate", mock.Anything, userID, staffID, mock.Anything).Return(nil)
	repo.On("GetByUserID", mock.Anything, userID).Return(complete, nil).Once()
	repo.On("GetAccountBasics", mock.Anything, userID).Return("Ada", "Lovelace", "0414", "a@b.c", nil)
	repo.On("HasRiderRole", mock.Anything, userID).Return(true, nil)
	stream.On("PublisherEvent", mock.Anything, models.DataEventRiderUpdated{
		MessageType: "rider_updated",
		RiderID:     userID.String(),
	}).Return(nil)

	resp, err := svc.Activate(context.Background(), userID, staffID)

	require.NoError(t, err)
	assert.Equal(t, userID.String(), resp.UserID)
}

func TestActivate_IncompleteProfile(t *testing.T) {
	repo := mocksriders.NewRiderProfileRepo(t)
	svc := app.NewRiderProfileApp(repo, mockuploadimage.NewUploadImage(t), nil)

	userID, staffID := uuid.New(), uuid.New()
	repo.On("GetByUserID", mock.Anything, userID).Return(&models.RiderProfile{UserID: userID, Status: models.RiderProfileDraft}, nil)

	_, err := svc.Activate(context.Background(), userID, staffID)

	assert.ErrorIs(t, err, errcustom.ErrRiderProfileIncomplete)
}

func TestSuspend_Success(t *testing.T) {
	repo := mocksriders.NewRiderProfileRepo(t)
	stream := mocksriders.NewRidersStream(t)
	svc := app.NewRiderProfileApp(repo, mockuploadimage.NewUploadImage(t), stream)

	userID := uuid.New()
	repo.On("Suspend", mock.Anything, userID).Return(nil)
	repo.On("DisableRiderRole", mock.Anything, userID).Return(nil)
	repo.On("GetByUserID", mock.Anything, userID).Return(&models.RiderProfile{UserID: userID, Status: models.RiderProfileSuspended}, nil)
	repo.On("GetAccountBasics", mock.Anything, userID).Return("Ada", "Lovelace", "0414", "a@b.c", nil)
	repo.On("HasRiderRole", mock.Anything, userID).Return(false, nil)
	stream.On("PublisherEvent", mock.Anything, models.DataEventRiderUpdated{
		MessageType: "rider_updated",
		RiderID:     userID.String(),
	}).Return(nil)

	resp, err := svc.Suspend(context.Background(), userID)

	require.NoError(t, err)
	assert.Equal(t, string(models.RiderProfileSuspended), resp.Status)
}

func TestSuspend_RepoError(t *testing.T) {
	repo := mocksriders.NewRiderProfileRepo(t)
	svc := app.NewRiderProfileApp(repo, mockuploadimage.NewUploadImage(t), nil)

	userID := uuid.New()
	repo.On("Suspend", mock.Anything, userID).Return(errcustom.ErrUnexpectedError)

	_, err := svc.Suspend(context.Background(), userID)

	assert.ErrorIs(t, err, errcustom.ErrUnexpectedError)
}

func TestTouchContact_Success(t *testing.T) {
	repo := mocksriders.NewRiderProfileRepo(t)
	stream := mocksriders.NewRidersStream(t)
	svc := app.NewRiderProfileApp(repo, mockuploadimage.NewUploadImage(t), stream)

	userID := uuid.New()
	notes := "  llamó, no contesta  "
	repo.On("TouchContact", mock.Anything, userID, mock.Anything, mock.MatchedBy(func(n *string) bool {
		return n != nil && *n == "llamó, no contesta"
	})).Return(nil)
	repo.On("GetByUserID", mock.Anything, userID).Return(&models.RiderProfile{UserID: userID, Status: models.RiderProfileDraft}, nil)
	repo.On("GetAccountBasics", mock.Anything, userID).Return("Ada", "Lovelace", "0414", "a@b.c", nil)
	repo.On("HasRiderRole", mock.Anything, userID).Return(false, nil)
	stream.On("PublisherEvent", mock.Anything, models.DataEventRiderUpdated{
		MessageType: "rider_updated",
		RiderID:     userID.String(),
	}).Return(nil)

	_, err := svc.TouchContact(context.Background(), userID, &notes)

	require.NoError(t, err)
}

func TestIsActiveRider_True(t *testing.T) {
	repo := mocksriders.NewRiderProfileRepo(t)
	svc := app.NewRiderProfileApp(repo, mockuploadimage.NewUploadImage(t), nil)

	userID := uuid.New()
	repo.On("GetByUserID", mock.Anything, userID).Return(&models.RiderProfile{UserID: userID, Status: models.RiderProfileActive}, nil)

	active, err := svc.IsActiveRider(context.Background(), userID)

	require.NoError(t, err)
	assert.True(t, active)
}

func TestIsActiveRider_LegacyWithoutProfile(t *testing.T) {
	repo := mocksriders.NewRiderProfileRepo(t)
	svc := app.NewRiderProfileApp(repo, mockuploadimage.NewUploadImage(t), nil)

	userID := uuid.New()
	repo.On("GetByUserID", mock.Anything, userID).Return(nil, errcustom.ErrNotFound)

	active, err := svc.IsActiveRider(context.Background(), userID)

	require.NoError(t, err)
	assert.True(t, active)
}

func TestIsActiveRider_RepoError(t *testing.T) {
	repo := mocksriders.NewRiderProfileRepo(t)
	svc := app.NewRiderProfileApp(repo, mockuploadimage.NewUploadImage(t), nil)

	userID := uuid.New()
	repo.On("GetByUserID", mock.Anything, userID).Return(nil, errcustom.ErrUnexpectedError)

	_, err := svc.IsActiveRider(context.Background(), userID)

	assert.ErrorIs(t, err, errcustom.ErrUnexpectedError)
}

// completeProfile arma un perfil con todos los campos requeridos por
// RiderProfile.Completion() para que Activate no falle por incompletitud.
func completeProfile(userID uuid.UUID, vehicleType string) *models.RiderProfile {
	url := "https://cdn/x.jpg"
	plate := "AB123CD"
	return &models.RiderProfile{
		UserID:             userID,
		Status:             models.RiderProfileDraft,
		ProfilePhotoURL:    &url,
		IDPhotoURL:         &url,
		CarnetPhotoURL:     &url,
		LicensePhotoURL:    &url,
		CirculationCardURL: &url,
		VehiclePhoto1URL:   &url,
		VehiclePhoto2URL:   &url,
		VehiclePhoto3URL:   &url,
		VehiclePhoto4URL:   &url,
		VehicleType:        &vehicleType,
		VehiclePlate:       &plate,
	}
}
