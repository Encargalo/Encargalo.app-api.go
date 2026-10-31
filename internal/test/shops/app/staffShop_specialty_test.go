package app_test

import (
	"context"
	"testing"

	mocksshops "Encargalo.app-api.go/internal/mocks/shops/domain/ports"
	"Encargalo.app-api.go/internal/shops/app"
	"Encargalo.app-api.go/internal/shops/domain/dtos"
	shopsmodels "Encargalo.app-api.go/internal/shops/domain/models/shops"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCreateDraftShop_SetsSpecialty(t *testing.T) {
	repo := mocksshops.NewStaffShopRepo(t)
	staffApp := app.NewStaffShopApp(repo, nil, nil, nil)

	ownerID := uuid.New()
	req := dtos.CreateDraftShopRequest{Name: "Panadería El Trigo", Type: "Supermarket", Specialty: shopsmodels.ShopSpecialtyBakery, OwnerID: ownerID}

	repo.On("CreateDraftShop", context.Background(), mock.MatchedBy(func(s *shopsmodels.Shop) bool {
		return s.Specialty != nil && *s.Specialty == shopsmodels.ShopSpecialtyBakery
	})).Run(func(args mock.Arguments) {
		shop := args.Get(1).(*shopsmodels.Shop)
		shop.ID = uuid.New()
	}).Return(nil)
	repo.On("EnsureOwnerRole", context.Background(), ownerID, mock.AnythingOfType("uuid.UUID")).Return(nil)
	repo.On("GetShopByID", context.Background(), mock.AnythingOfType("uuid.UUID")).Return(&shopsmodels.Shop{
		OwnerID: ownerID, Name: "Panadería El Trigo", Type: "Supermarket", Status: shopsmodels.ShopStatusDraft,
	}, nil)

	_, err := staffApp.CreateDraftShop(context.Background(), req)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestCreateDraftShop_WithoutSpecialtyLeavesItNil(t *testing.T) {
	repo := mocksshops.NewStaffShopRepo(t)
	staffApp := app.NewStaffShopApp(repo, nil, nil, nil)

	ownerID := uuid.New()
	req := dtos.CreateDraftShopRequest{Name: "Super Genérico", Type: "Supermarket", OwnerID: ownerID}

	repo.On("CreateDraftShop", context.Background(), mock.MatchedBy(func(s *shopsmodels.Shop) bool {
		return s.Specialty == nil
	})).Run(func(args mock.Arguments) {
		shop := args.Get(1).(*shopsmodels.Shop)
		shop.ID = uuid.New()
	}).Return(nil)
	repo.On("EnsureOwnerRole", context.Background(), ownerID, mock.AnythingOfType("uuid.UUID")).Return(nil)
	repo.On("GetShopByID", context.Background(), mock.AnythingOfType("uuid.UUID")).Return(&shopsmodels.Shop{
		OwnerID: ownerID, Name: "Super Genérico", Type: "Supermarket", Status: shopsmodels.ShopStatusDraft,
	}, nil)

	_, err := staffApp.CreateDraftShop(context.Background(), req)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestUpdateDraftShop_SetsSpecialty(t *testing.T) {
	repo := mocksshops.NewStaffShopRepo(t)
	staffApp := app.NewStaffShopApp(repo, nil, nil, nil)

	shopID := uuid.New()
	existing := &shopsmodels.Shop{ID: shopID, Name: "Super Genérico", Type: "Supermarket"}

	repo.On("GetShopByID", context.Background(), shopID).Return(existing, nil).Once()
	repo.On("UpdateDraftShop", context.Background(), mock.MatchedBy(func(s *shopsmodels.Shop) bool {
		return s.Specialty != nil && *s.Specialty == shopsmodels.ShopSpecialtyProduce
	})).Return(nil)
	repo.On("GetShopByID", context.Background(), shopID).Return(existing, nil).Once()

	specialty := shopsmodels.ShopSpecialtyProduce
	_, err := staffApp.UpdateDraftShop(context.Background(), shopID, dtos.UpdateDraftShopRequest{
		Specialty: &specialty,
	})

	require.NoError(t, err)
	repo.AssertExpectations(t)
}
