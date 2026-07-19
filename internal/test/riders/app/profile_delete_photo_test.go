package app_test

import (
	"context"
	"testing"

	mocksriders "Encargalo.app-api.go/internal/mocks/riders/domain/ports"
	uploadimage "Encargalo.app-api.go/internal/pkg/uploadImage"
	"Encargalo.app-api.go/internal/riders/app"
	"Encargalo.app-api.go/internal/riders/domain/models"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type stubUploader struct {
	deleted []string
	err     error
}

func (s *stubUploader) UploadImage(ctx context.Context, imageB64 string) (*uploader.UploadResult, error) {
	return &uploader.UploadResult{SecureURL: "https://res.cloudinary.com/demo/image/upload/v1/x.jpg"}, nil
}

func (s *stubUploader) DeleteImage(ctx context.Context, secureURL string) error {
	s.deleted = append(s.deleted, secureURL)
	return s.err
}

var _ uploadimage.UploadImage = (*stubUploader)(nil)

func TestDeletePhoto_ClearsDBAndCloudinary(t *testing.T) {
	repo := mocksriders.NewRiderProfileRepo(t)
	up := &stubUploader{}
	svc := app.NewRiderProfileApp(repo, up)

	userID := uuid.New()
	url := "https://res.cloudinary.com/demo/image/upload/v1/folder/id.jpg"
	profile := &models.RiderProfile{
		UserID:     userID,
		Status:     models.RiderProfileDraft,
		IDPhotoURL: &url,
	}

	repo.On("GetByUserID", mock.Anything, userID).Return(profile, nil).Once()
	repo.On("ClearPhoto", mock.Anything, userID, "id_photo").Return(nil).Once()
	// GetProfile after clear
	cleared := &models.RiderProfile{UserID: userID, Status: models.RiderProfileDraft}
	repo.On("GetByUserID", mock.Anything, userID).Return(cleared, nil).Once()
	repo.On("GetAccountBasics", mock.Anything, userID).Return("Ada", "Lovelace", "+58000", "a@b.c", nil)
	repo.On("HasRiderRole", mock.Anything, userID).Return(false, nil)

	resp, err := svc.DeletePhoto(context.Background(), userID, "id_photo")
	require.NoError(t, err)
	assert.Equal(t, userID.String(), resp.UserID)
	assert.Equal(t, []string{url}, up.deleted)
}

func TestDeletePhoto_NoURL_Noop(t *testing.T) {
	repo := mocksriders.NewRiderProfileRepo(t)
	up := &stubUploader{}
	svc := app.NewRiderProfileApp(repo, up)

	userID := uuid.New()
	profile := &models.RiderProfile{UserID: userID, Status: models.RiderProfileDraft}

	repo.On("GetByUserID", mock.Anything, userID).Return(profile, nil).Once()
	repo.On("GetByUserID", mock.Anything, userID).Return(profile, nil).Once()
	repo.On("GetAccountBasics", mock.Anything, userID).Return("Ada", "Lovelace", "+58000", "a@b.c", nil)
	repo.On("HasRiderRole", mock.Anything, userID).Return(false, nil)

	_, err := svc.DeletePhoto(context.Background(), userID, "id_photo")
	require.NoError(t, err)
	assert.Empty(t, up.deleted)
}

func TestDeletePhoto_InvalidField(t *testing.T) {
	repo := mocksriders.NewRiderProfileRepo(t)
	svc := app.NewRiderProfileApp(repo, &stubUploader{})

	_, err := svc.DeletePhoto(context.Background(), uuid.New(), "nope")
	assert.ErrorIs(t, err, errcustom.ErrNotFound)
}

func TestDeletePhoto_CloudinaryFails_StillClearsDB(t *testing.T) {
	repo := mocksriders.NewRiderProfileRepo(t)
	up := &stubUploader{err: assert.AnError}
	svc := app.NewRiderProfileApp(repo, up)

	userID := uuid.New()
	url := "https://res.cloudinary.com/demo/image/upload/v1/folder/id.jpg"
	profile := &models.RiderProfile{
		UserID:     userID,
		Status:     models.RiderProfileDraft,
		IDPhotoURL: &url,
	}

	repo.On("GetByUserID", mock.Anything, userID).Return(profile, nil).Once()
	repo.On("ClearPhoto", mock.Anything, userID, "id_photo").Return(nil).Once()
	cleared := &models.RiderProfile{UserID: userID, Status: models.RiderProfileDraft}
	repo.On("GetByUserID", mock.Anything, userID).Return(cleared, nil).Once()
	repo.On("GetAccountBasics", mock.Anything, userID).Return("Ada", "Lovelace", "+58000", "a@b.c", nil)
	repo.On("HasRiderRole", mock.Anything, userID).Return(false, nil)

	_, err := svc.DeletePhoto(context.Background(), userID, "id_photo")
	require.NoError(t, err)
}
