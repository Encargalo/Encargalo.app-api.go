package app_test

import (
	"context"
	"testing"

	mocksshops "Encargalo.app-api.go/internal/mocks/shops/domain/ports"
	jsonutil "Encargalo.app-api.go/internal/pkg/json"
	uploadimage "Encargalo.app-api.go/internal/pkg/uploadImage"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/shops/app"
	"Encargalo.app-api.go/internal/shops/domain/dtos"
	shopsmodels "Encargalo.app-api.go/internal/shops/domain/models/shops"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// stubUploader simula Cloudinary devolviendo siempre la misma SecureURL,
// sin hacer red — igual patrón que internal/test/riders/app/profile_delete_photo_test.go.
type stubUploader struct {
	secureURL string
}

func (s *stubUploader) UploadImage(ctx context.Context, imageB64 string) (*uploader.UploadResult, error) {
	return &uploader.UploadResult{SecureURL: s.secureURL}, nil
}

func (s *stubUploader) DeleteImage(ctx context.Context, secureURL string) error {
	return nil
}

var _ uploadimage.UploadImage = (*stubUploader)(nil)

func TestCreateDraftShop_CreatesAndLinksOwner(t *testing.T) {
	repo := mocksshops.NewStaffShopRepo(t)
	staffApp := app.NewStaffShopApp(repo, nil, nil)

	ownerID := uuid.New()
	req := dtos.CreateDraftShopRequest{Name: "Pizzería Don Luigi", Type: "Restaurant", OwnerID: ownerID}

	repo.On("CreateDraftShop", context.Background(), mock.MatchedBy(func(s *shopsmodels.Shop) bool {
		return s.Name == "Pizzería Don Luigi" && s.Type == "Restaurant" && s.OwnerID == ownerID && s.Status == shopsmodels.ShopStatusDraft
	})).Run(func(args mock.Arguments) {
		shop := args.Get(1).(*shopsmodels.Shop)
		shop.ID = uuid.New()
	}).Return(nil)
	repo.On("EnsureOwnerRole", context.Background(), ownerID, mock.AnythingOfType("uuid.UUID")).Return(nil)
	repo.On("GetShopByID", context.Background(), mock.AnythingOfType("uuid.UUID")).Return(&shopsmodels.Shop{
		OwnerID: ownerID, Name: "Pizzería Don Luigi", Type: "Restaurant", Status: shopsmodels.ShopStatusDraft,
	}, nil)

	resp, err := staffApp.CreateDraftShop(context.Background(), req)

	require.NoError(t, err)
	require.Equal(t, "draft", resp.Status)
	require.Equal(t, ownerID, resp.OwnerID)
	repo.AssertExpectations(t)
}

func TestUpdateDraftShop_RejectsWhenShopIsOpen(t *testing.T) {
	repo := mocksshops.NewStaffShopRepo(t)
	staffApp := app.NewStaffShopApp(repo, nil, nil)

	shopID := uuid.New()
	existing := &shopsmodels.Shop{ID: shopID, Name: "Pizzería Don Luigi", Type: "Restaurant", Opened: true}

	repo.On("GetShopByID", context.Background(), shopID).Return(existing, nil).Once()

	_, err := staffApp.UpdateDraftShop(context.Background(), shopID, dtos.UpdateDraftShopRequest{})

	require.ErrorIs(t, err, errcustom.ErrShopOpen)
	repo.AssertExpectations(t)
}

func TestUpdateDraftShop_PartialUpdatesDoNotClobberExistingFields(t *testing.T) {
	repo := mocksshops.NewStaffShopRepo(t)
	uploaderURL := "https://res.cloudinary.com/demo/image/upload/v1/shops/logo.png"
	staffApp := app.NewStaffShopApp(repo, nil, &stubUploader{secureURL: uploaderURL})

	shopID := uuid.New()
	existing := &shopsmodels.Shop{ID: shopID, Name: "Pizzería Don Luigi", Type: "Restaurant", Address: "Dirección vieja"}

	repo.On("GetShopByID", context.Background(), shopID).Return(existing, nil).Once()
	logo := "data:image/png;base64,AAA"
	repo.On("UpdateDraftShop", context.Background(), mock.MatchedBy(func(s *shopsmodels.Shop) bool {
		return s.LogoImage == uploaderURL && s.Address == "Dirección vieja"
	})).Return(nil)
	repo.On("GetShopByID", context.Background(), shopID).Return(existing, nil).Once()

	_, err := staffApp.UpdateDraftShop(context.Background(), shopID, dtos.UpdateDraftShopRequest{
		LogoImage: jsonutil.Nullable[string]{Set: true, Valid: true, Value: logo},
	})

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestUpdateDraftShop_LogoImageUploadFailureIsPropagated(t *testing.T) {
	repo := mocksshops.NewStaffShopRepo(t)
	staffApp := app.NewStaffShopApp(repo, nil, nil)

	shopID := uuid.New()
	existing := &shopsmodels.Shop{ID: shopID, Name: "Pizzería Don Luigi", Type: "Restaurant"}

	repo.On("GetShopByID", context.Background(), shopID).Return(existing, nil).Once()

	_, err := staffApp.UpdateDraftShop(context.Background(), shopID, dtos.UpdateDraftShopRequest{
		LogoImage: jsonutil.Nullable[string]{Set: true, Valid: true, Value: "data:image/png;base64,AAA"},
	})

	require.ErrorIs(t, err, errcustom.ErrUploadingImage)
	repo.AssertNotCalled(t, "UpdateDraftShop", mock.Anything, mock.Anything)
}

func TestUpdateDraftShop_ExplicitNullClearsField(t *testing.T) {
	repo := mocksshops.NewStaffShopRepo(t)
	staffApp := app.NewStaffShopApp(repo, nil, nil)

	shopID := uuid.New()
	lat, lon := 10.4806, -66.9036
	existing := &shopsmodels.Shop{
		ID: shopID, Name: "Pizzería Don Luigi", Type: "Restaurant",
		Latitude: &lat, Longitude: &lon, LogoImage: "logo", Banner: "banner",
	}

	repo.On("GetShopByID", context.Background(), shopID).Return(existing, nil).Once()
	repo.On("UpdateDraftShop", context.Background(), mock.MatchedBy(func(s *shopsmodels.Shop) bool {
		return s.Latitude == nil && s.Longitude == nil && s.LogoImage == "" && s.Banner == ""
	})).Return(nil)
	repo.On("GetShopByID", context.Background(), shopID).Return(existing, nil).Once()

	_, err := staffApp.UpdateDraftShop(context.Background(), shopID, dtos.UpdateDraftShopRequest{
		Latitude:  jsonutil.Nullable[float64]{Set: true, Valid: false},
		Longitude: jsonutil.Nullable[float64]{Set: true, Valid: false},
		LogoImage: jsonutil.Nullable[string]{Set: true, Valid: false},
		Banner:    jsonutil.Nullable[string]{Set: true, Valid: false},
	})

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestUpdateDraftShop_OmittedFieldsAreNotTouched(t *testing.T) {
	repo := mocksshops.NewStaffShopRepo(t)
	staffApp := app.NewStaffShopApp(repo, nil, nil)

	shopID := uuid.New()
	lat, lon := 10.4806, -66.9036
	existing := &shopsmodels.Shop{
		ID: shopID, Name: "Pizzería Don Luigi", Type: "Restaurant",
		Latitude: &lat, Longitude: &lon, LogoImage: "logo", Banner: "banner",
	}

	repo.On("GetShopByID", context.Background(), shopID).Return(existing, nil).Once()
	repo.On("UpdateDraftShop", context.Background(), mock.MatchedBy(func(s *shopsmodels.Shop) bool {
		return s.Latitude == &lat && s.Longitude == &lon && s.LogoImage == "logo" && s.Banner == "banner"
	})).Return(nil)
	repo.On("GetShopByID", context.Background(), shopID).Return(existing, nil).Once()

	_, err := staffApp.UpdateDraftShop(context.Background(), shopID, dtos.UpdateDraftShopRequest{})

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestActivateShop_IncompleteReturnsError(t *testing.T) {
	repo := mocksshops.NewStaffShopRepo(t)
	staffApp := app.NewStaffShopApp(repo, nil, nil)

	shopID, staffID := uuid.New(), uuid.New()
	repo.On("GetShopByID", context.Background(), shopID).Return(&shopsmodels.Shop{
		ID: shopID, Name: "Pizzería Don Luigi", Type: "Restaurant", Status: shopsmodels.ShopStatusDraft,
	}, nil)

	_, err := staffApp.ActivateShop(context.Background(), shopID, staffID)

	require.ErrorIs(t, err, errcustom.ErrShopOnboardingIncomplete)
	repo.AssertNotCalled(t, "ActivateShop", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestActivateShop_CompleteActivates(t *testing.T) {
	repo := mocksshops.NewStaffShopRepo(t)
	staffApp := app.NewStaffShopApp(repo, nil, nil)

	shopID, staffID := uuid.New(), uuid.New()
	lat, lon := 10.4806, -66.9036
	completeShop := &shopsmodels.Shop{
		ID: shopID, Name: "Pizzería Don Luigi", Type: "Restaurant", Status: shopsmodels.ShopStatusDraft,
		Slug: "don-luigi", LogoImage: "logo", Banner: "banner", Address: "dir", HomePhone: "+58...",
		Latitude: &lat, Longitude: &lon,
	}
	repo.On("GetShopByID", context.Background(), shopID).Return(completeShop, nil).Once()
	repo.On("ActivateShop", context.Background(), shopID, staffID, mock.AnythingOfType("time.Time")).Return(nil)
	repo.On("GetShopByID", context.Background(), shopID).Return(&shopsmodels.Shop{
		ID: shopID, Status: shopsmodels.ShopStatusActive,
	}, nil).Once()

	resp, err := staffApp.ActivateShop(context.Background(), shopID, staffID)

	require.NoError(t, err)
	require.Equal(t, "active", resp.Status)
	repo.AssertExpectations(t)
}

func TestActivateShop_AlreadyActiveIsIdempotent(t *testing.T) {
	repo := mocksshops.NewStaffShopRepo(t)
	staffApp := app.NewStaffShopApp(repo, nil, nil)

	shopID, staffID := uuid.New(), uuid.New()
	repo.On("GetShopByID", context.Background(), shopID).Return(&shopsmodels.Shop{
		ID: shopID, Status: shopsmodels.ShopStatusActive,
	}, nil)

	resp, err := staffApp.ActivateShop(context.Background(), shopID, staffID)

	require.NoError(t, err)
	require.Equal(t, "active", resp.Status)
	repo.AssertNotCalled(t, "ActivateShop", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}
