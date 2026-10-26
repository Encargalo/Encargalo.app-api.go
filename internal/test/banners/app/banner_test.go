package app_test

import (
	"context"
	"testing"

	"Encargalo.app-api.go/internal/banners/app"
	"Encargalo.app-api.go/internal/banners/domain/dtos"
	"Encargalo.app-api.go/internal/banners/domain/models"
	mocksbanners "Encargalo.app-api.go/internal/mocks/banners/domain/ports"
	jsonutil "Encargalo.app-api.go/internal/pkg/json"
	uploadimage "Encargalo.app-api.go/internal/pkg/uploadImage"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// stubUploader simula Cloudinary sin hacer red, igual patrón que
// internal/test/shops/app/staffShop_test.go.
type stubUploader struct {
	secureURL  string
	uploadErr  error
	deletedURL string
}

func (s *stubUploader) UploadImage(ctx context.Context, imageB64 string) (*uploader.UploadResult, error) {
	if s.uploadErr != nil {
		return nil, s.uploadErr
	}
	return &uploader.UploadResult{SecureURL: s.secureURL}, nil
}

func (s *stubUploader) DeleteImage(ctx context.Context, secureURL string) error {
	s.deletedURL = secureURL
	return nil
}

var _ uploadimage.UploadImage = (*stubUploader)(nil)

func TestCreateBanner_UploadsImageAndPersists(t *testing.T) {
	repo := mocksbanners.NewBannersRepo(t)
	imageURL := "https://res.cloudinary.com/demo/image/upload/v1/banners/a.webp"
	bannersApp := app.NewBannersApp(repo, &stubUploader{secureURL: imageURL})

	position := 2
	link := "https://encargalo.app/promo"
	req := dtos.CreateBannerRequest{Image: "data:image/png;base64,AAA", LinkURL: &link, Position: &position}

	repo.On("CreateBanner", context.Background(), mock.MatchedBy(func(b *models.Banner) bool {
		return b.ImageURL == imageURL && b.LinkURL != nil && *b.LinkURL == link && b.Position == 2 && b.Active
	})).Run(func(args mock.Arguments) {
		b := args.Get(1).(*models.Banner)
		b.ID = uuid.New()
	}).Return(nil)

	resp, err := bannersApp.CreateBanner(context.Background(), req)

	require.NoError(t, err)
	require.Equal(t, imageURL, resp.ImageURL)
	require.True(t, resp.Active)
	repo.AssertExpectations(t)
}

func TestCreateBanner_UploadFailureIsPropagated(t *testing.T) {
	repo := mocksbanners.NewBannersRepo(t)
	bannersApp := app.NewBannersApp(repo, nil)

	_, err := bannersApp.CreateBanner(context.Background(), dtos.CreateBannerRequest{Image: "data:image/png;base64,AAA"})

	require.ErrorIs(t, err, errcustom.ErrUploadingImage)
	repo.AssertNotCalled(t, "CreateBanner", mock.Anything, mock.Anything)
}

func TestUpdateBanner_ReplacesImageAndDeletesPrevious(t *testing.T) {
	repo := mocksbanners.NewBannersRepo(t)
	newImageURL := "https://res.cloudinary.com/demo/image/upload/v1/banners/new.webp"
	uploaderStub := &stubUploader{secureURL: newImageURL}
	bannersApp := app.NewBannersApp(repo, uploaderStub)

	bannerID := uuid.New()
	oldImageURL := "https://res.cloudinary.com/demo/image/upload/v1/banners/old.webp"
	existing := &models.Banner{ID: bannerID, ImageURL: oldImageURL, Position: 1, Active: true}

	repo.On("GetBannerByID", context.Background(), bannerID).Return(existing, nil).Once()
	repo.On("UpdateBanner", context.Background(), mock.MatchedBy(func(b *models.Banner) bool {
		return b.ImageURL == newImageURL
	})).Return(nil)

	newImage := "data:image/png;base64,BBB"
	_, err := bannersApp.UpdateBanner(context.Background(), bannerID, dtos.UpdateBannerRequest{Image: &newImage})

	require.NoError(t, err)
	require.Equal(t, oldImageURL, uploaderStub.deletedURL)
	repo.AssertExpectations(t)
}

func TestUpdateBanner_OmittedFieldsAreNotTouched(t *testing.T) {
	repo := mocksbanners.NewBannersRepo(t)
	bannersApp := app.NewBannersApp(repo, nil)

	bannerID := uuid.New()
	link := "https://encargalo.app/promo"
	existing := &models.Banner{ID: bannerID, ImageURL: "img", LinkURL: &link, Position: 3, Active: true}

	repo.On("GetBannerByID", context.Background(), bannerID).Return(existing, nil).Once()
	repo.On("UpdateBanner", context.Background(), mock.MatchedBy(func(b *models.Banner) bool {
		return b.ImageURL == "img" && b.LinkURL != nil && *b.LinkURL == link && b.Position == 3 && b.Active
	})).Return(nil)

	_, err := bannersApp.UpdateBanner(context.Background(), bannerID, dtos.UpdateBannerRequest{})

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestUpdateBanner_ExplicitNullClearsLinkURL(t *testing.T) {
	repo := mocksbanners.NewBannersRepo(t)
	bannersApp := app.NewBannersApp(repo, nil)

	bannerID := uuid.New()
	link := "https://encargalo.app/promo"
	existing := &models.Banner{ID: bannerID, ImageURL: "img", LinkURL: &link, Active: true}

	repo.On("GetBannerByID", context.Background(), bannerID).Return(existing, nil).Once()
	repo.On("UpdateBanner", context.Background(), mock.MatchedBy(func(b *models.Banner) bool {
		return b.LinkURL == nil
	})).Return(nil)

	_, err := bannersApp.UpdateBanner(context.Background(), bannerID, dtos.UpdateBannerRequest{
		LinkURL: jsonutil.Nullable[string]{Set: true, Valid: false},
	})

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestUpdateBanner_CanDeactivate(t *testing.T) {
	repo := mocksbanners.NewBannersRepo(t)
	bannersApp := app.NewBannersApp(repo, nil)

	bannerID := uuid.New()
	existing := &models.Banner{ID: bannerID, ImageURL: "img", Active: true}
	inactive := false

	repo.On("GetBannerByID", context.Background(), bannerID).Return(existing, nil).Once()
	repo.On("UpdateBanner", context.Background(), mock.MatchedBy(func(b *models.Banner) bool {
		return !b.Active
	})).Return(nil)

	_, err := bannersApp.UpdateBanner(context.Background(), bannerID, dtos.UpdateBannerRequest{Active: &inactive})

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestDeleteBanner_DeletesRowAndImage(t *testing.T) {
	repo := mocksbanners.NewBannersRepo(t)
	uploaderStub := &stubUploader{}
	bannersApp := app.NewBannersApp(repo, uploaderStub)

	bannerID := uuid.New()
	imageURL := "https://res.cloudinary.com/demo/image/upload/v1/banners/a.webp"
	repo.On("GetBannerByID", context.Background(), bannerID).Return(&models.Banner{ID: bannerID, ImageURL: imageURL}, nil)
	repo.On("DeleteBanner", context.Background(), bannerID).Return(nil)

	err := bannersApp.DeleteBanner(context.Background(), bannerID)

	require.NoError(t, err)
	require.Equal(t, imageURL, uploaderStub.deletedURL)
	repo.AssertExpectations(t)
}

func TestListActiveBanners_MapsToDTO(t *testing.T) {
	repo := mocksbanners.NewBannersRepo(t)
	bannersApp := app.NewBannersApp(repo, nil)

	repo.On("ListActiveBanners", context.Background()).Return(models.Banners{
		{ID: uuid.New(), ImageURL: "img1", Active: true},
	}, nil)

	resp, err := bannersApp.ListActiveBanners(context.Background())

	require.NoError(t, err)
	require.Len(t, resp, 1)
	require.Equal(t, "img1", resp[0].ImageURL)
}
