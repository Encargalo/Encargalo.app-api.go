package app_test

import (
	"context"
	"testing"

	mocksshops "Encargalo.app-api.go/internal/mocks/shops/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/shops/app"
	shopsmodels "Encargalo.app-api.go/internal/shops/domain/models/shops"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestFindOwnerCandidate_Success(t *testing.T) {
	repo := mocksshops.NewStaffShopRepo(t)
	staffApp := app.NewStaffShopApp(repo, nil, nil)

	userID := uuid.New()
	repo.On("FindOwnerCandidateByPhone", context.Background(), "04141234567").
		Return(userID, "Luigi", "Verdi", "04141234567", "luigi@example.com", nil)
	repo.On("HasAnyOwnerRole", context.Background(), userID).Return(true, nil)

	resp, err := staffApp.FindOwnerCandidate(context.Background(), " 04141234567 ")

	require.NoError(t, err)
	require.Equal(t, userID.String(), resp.UserID)
	require.True(t, resp.HasOwnerRole)
}

func TestFindOwnerCandidate_NotFound(t *testing.T) {
	repo := mocksshops.NewStaffShopRepo(t)
	staffApp := app.NewStaffShopApp(repo, nil, nil)

	repo.On("FindOwnerCandidateByPhone", context.Background(), "0000").
		Return(uuid.Nil, "", "", "", "", errcustom.ErrNotFound)

	_, err := staffApp.FindOwnerCandidate(context.Background(), "0000")

	require.ErrorIs(t, err, errcustom.ErrNotFound)
}

func TestFindOwnerCandidate_HasAnyOwnerRoleError(t *testing.T) {
	repo := mocksshops.NewStaffShopRepo(t)
	staffApp := app.NewStaffShopApp(repo, nil, nil)

	userID := uuid.New()
	repo.On("FindOwnerCandidateByPhone", context.Background(), "0414").
		Return(userID, "Luigi", "Verdi", "0414", "luigi@example.com", nil)
	repo.On("HasAnyOwnerRole", context.Background(), userID).Return(false, errcustom.ErrUnexpectedError)

	_, err := staffApp.FindOwnerCandidate(context.Background(), "0414")

	require.ErrorIs(t, err, errcustom.ErrUnexpectedError)
}

func TestListShops_DefaultsToDraftStatus(t *testing.T) {
	repo := mocksshops.NewStaffShopRepo(t)
	staffApp := app.NewStaffShopApp(repo, nil, nil)

	repo.On("ListShopsByStatus", context.Background(), shopsmodels.ShopStatusDraft, 1, 10).
		Return(shopsmodels.Shops{{ID: uuid.New(), Name: "Pizzería Don Luigi"}}, 1, nil)

	resp, err := staffApp.ListShops(context.Background(), "", 1, 10)

	require.NoError(t, err)
	require.Equal(t, 1, resp.Total)
	require.Len(t, resp.Shops, 1)
}

func TestListShops_PropagatesExplicitStatus(t *testing.T) {
	repo := mocksshops.NewStaffShopRepo(t)
	staffApp := app.NewStaffShopApp(repo, nil, nil)

	repo.On("ListShopsByStatus", context.Background(), shopsmodels.ShopStatusActive, 2, 5).
		Return(shopsmodels.Shops{}, 0, nil)

	resp, err := staffApp.ListShops(context.Background(), shopsmodels.ShopStatusActive, 2, 5)

	require.NoError(t, err)
	require.Equal(t, 0, resp.Total)
	require.Empty(t, resp.Shops)
}

func TestListShops_RepoError(t *testing.T) {
	repo := mocksshops.NewStaffShopRepo(t)
	staffApp := app.NewStaffShopApp(repo, nil, nil)

	repo.On("ListShopsByStatus", context.Background(), shopsmodels.ShopStatusDraft, 1, 10).
		Return(nil, 0, errcustom.ErrUnexpectedError)

	_, err := staffApp.ListShops(context.Background(), "", 1, 10)

	require.ErrorIs(t, err, errcustom.ErrUnexpectedError)
}

// ActivateShop propaga el error del repo cuando el shop está completo y la
// activación en sí falla (no cubierto por los tests existentes, que solo
// cubren incompleto/activo-idempotente/éxito).
func TestActivateShop_RepoErrorOnActivate(t *testing.T) {
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
	repo.On("ActivateShop", context.Background(), shopID, staffID, mock.AnythingOfType("time.Time")).Return(errcustom.ErrUnexpectedError)

	_, err := staffApp.ActivateShop(context.Background(), shopID, staffID)

	require.ErrorIs(t, err, errcustom.ErrUnexpectedError)
}
