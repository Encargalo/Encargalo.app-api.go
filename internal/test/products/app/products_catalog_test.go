package app_test

import (
	"context"
	"testing"
	"time"

	bcvmodels "Encargalo.app-api.go/internal/bcv/domain/models"
	"Encargalo.app-api.go/internal/products/domain/dtos"
	"Encargalo.app-api.go/internal/products/domain/models"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestSearchProductsByShopID_FiltersScheduleAppliesLaunchPromoAndRate(t *testing.T) {
	svc, m := newProductAppFixture(t)
	shopID := uuid.New()

	visible := dtos.CategoryResponse{
		Name: "Visible",
		Items: dtos.ItemsResponse{
			{Ref: 10, EffectiveRef: 10, Details: dtos.ItemDetailsResponse{Additions: dtos.AdditionsResponse{{Ref: 2}}}},
		},
	}
	hidden := dtos.CategoryResponse{
		Name:      "Fuera de horario",
		Schedules: []dtos.ScheduleResponse{{StartTime: "00:00:00", EndTime: "23:59:59", Enabled: false}},
	}
	data := dtos.CategoriesResponse{visible, hidden}

	m.repo.On("SearchProductsByShopID", mock.Anything, shopID, true).Return(data, nil)
	endsAt := time.Now().Add(time.Hour)
	m.promo.On("GetLaunchPromoEndsAt", mock.Anything, shopID).Return(&endsAt, nil)
	m.bcv.On("GetRate", mock.Anything).Return(&bcvmodels.ExchangeRate{Rate: 40}, nil)

	got, err := svc.SearchProductsByShopID(context.Background(), shopID, true)

	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "Visible", got[0].Name)
	item := got[0].Items[0]
	// EffectiveRef 10 -> 50% launch promo -> 5, luego Bs a tasa 40.
	assert.Equal(t, 5.0, item.EffectiveRef)
	assert.True(t, item.HasLaunchPromo)
	assert.Equal(t, 400.0, item.Bs)
	assert.Equal(t, 200.0, item.EffectiveBs)
	assert.Equal(t, 80.0, item.Details.Additions[0].Bs)
}

func TestSearchProductsByShopID_RepoError(t *testing.T) {
	svc, m := newProductAppFixture(t)
	shopID := uuid.New()

	m.repo.On("SearchProductsByShopID", mock.Anything, shopID, false).Return(nil, assert.AnError)

	_, err := svc.SearchProductsByShopID(context.Background(), shopID, false)

	assert.ErrorIs(t, err, assert.AnError)
}

func TestSearchProductsByShopIDForOwner_NoScheduleFilterNoLaunchPromo(t *testing.T) {
	svc, m := newProductAppFixture(t)
	shopID := uuid.New()

	data := dtos.CategoriesResponse{{
		Name:      "Fuera de horario pero visible para el dueño",
		Schedules: []dtos.ScheduleResponse{{StartTime: "00:00:00", EndTime: "23:59:59", Enabled: false}},
		Items:     dtos.ItemsResponse{{Ref: 10, EffectiveRef: 10}},
	}}
	m.repo.On("SearchProductsByShopID", mock.Anything, shopID, false).Return(data, nil)
	m.bcv.On("GetRate", mock.Anything).Return(&bcvmodels.ExchangeRate{Rate: 40}, nil)

	got, err := svc.SearchProductsByShopIDForOwner(context.Background(), shopID)

	require.NoError(t, err)
	require.Len(t, got, 1)
	// Sin promo aplicada: EffectiveRef se mantiene en 10 (Bs = 400).
	assert.Equal(t, 400.0, got[0].Items[0].EffectiveBs)
	m.promo.AssertNotCalled(t, "GetLaunchPromoEndsAt", mock.Anything, mock.Anything)
}

func TestSearchAdditionsByItemID_AppliesRate(t *testing.T) {
	svc, m := newProductAppFixture(t)
	itemID := uuid.New()

	m.repo.On("SearchAdditionsByItemID", mock.Anything, itemID).Return(dtos.AdditionsResponse{{Ref: 3}}, nil)
	m.bcv.On("GetRate", mock.Anything).Return(&bcvmodels.ExchangeRate{Rate: 40}, nil)

	got, err := svc.SearchAdditionsByItemID(context.Background(), itemID)

	require.NoError(t, err)
	assert.Equal(t, 120.0, got[0].Bs)
}

func TestSearchAdditionsCatalogByShopID_AppliesRate(t *testing.T) {
	svc, m := newProductAppFixture(t)
	shopID := uuid.New()

	m.repo.On("SearchAdditionsCatalogByShopID", mock.Anything, shopID).Return(dtos.AdditionsResponse{{Ref: 3}}, nil)
	m.bcv.On("GetRate", mock.Anything).Return(&bcvmodels.ExchangeRate{Rate: 40}, nil)

	got, err := svc.SearchAdditionsCatalogByShopID(context.Background(), shopID)

	require.NoError(t, err)
	assert.Equal(t, 120.0, got[0].Bs)
}

func TestSearchFlavorsByItemID_DelegatesToRepo(t *testing.T) {
	svc, m := newProductAppFixture(t)
	itemID := uuid.New()
	expected := dtos.FlavorsResponse{{ID: uuid.New(), Name: "Picante"}}
	m.repo.On("SearchFlavorsByItemID", mock.Anything, itemID).Return(expected, nil)

	got, err := svc.SearchFlavorsByItemID(context.Background(), itemID)

	require.NoError(t, err)
	assert.Equal(t, expected, got)
}

func TestSearchFlavorsByCategoryID_DelegatesToRepo(t *testing.T) {
	svc, m := newProductAppFixture(t)
	categoryID := uuid.New()
	expected := dtos.FlavorsResponse{{ID: uuid.New(), Name: "Dulce"}}
	m.repo.On("SearchFlavorsByCategoryID", mock.Anything, categoryID).Return(expected, nil)

	got, err := svc.SearchFlavorsByCategoryID(context.Background(), categoryID)

	require.NoError(t, err)
	assert.Equal(t, expected, got)
}

func TestSearchBestSellersByShopID_FiltersScheduleAndAppliesRate(t *testing.T) {
	svc, m := newProductAppFixture(t)
	shopID := uuid.New()

	visible := dtos.ItemResponse{Name: "Visible", Ref: 10, EffectiveRef: 10}
	hidden := dtos.ItemResponse{
		Name:      "Fuera de horario",
		Schedules: []dtos.ScheduleResponse{{StartTime: "00:00:00", EndTime: "23:59:59", Enabled: false}},
	}
	m.repo.On("SearchBestSellersByShopID", mock.Anything, shopID).Return(dtos.ItemsResponse{visible, hidden}, nil)
	m.promo.On("GetLaunchPromoEndsAt", mock.Anything, shopID).Return(nil, nil)
	m.bcv.On("GetRate", mock.Anything).Return(&bcvmodels.ExchangeRate{Rate: 40}, nil)

	got, err := svc.SearchBestSellersByShopID(context.Background(), shopID)

	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "Visible", got[0].Name)
	assert.Equal(t, 400.0, got[0].Bs)
}

func TestAddSoldItem_DelegatesToRepo(t *testing.T) {
	svc, m := newProductAppFixture(t)
	item := models.DataProduceOrder{ProductID: uuid.New(), Quantity: 2}
	m.repo.On("AddSoldItem", mock.Anything, item).Return(nil)

	err := svc.AddSoldItem(context.Background(), item)

	assert.NoError(t, err)
}

func TestGetTrendingByCoords_FiltersScheduleAndLimitsTo10(t *testing.T) {
	svc, m := newProductAppFixture(t)
	coords := dtos.TrendingCoords{Latitude: 10, Longitude: -66}

	entries := make(dtos.TrendingResponse, 12)
	for i := range entries {
		entries[i] = dtos.TrendingEntry{
			Product: dtos.TrendingProduct{Name: "p", Ref: 10, EffectiveRef: 10},
			Shop:    dtos.TrendingShop{DeliveryFee: 2},
		}
	}
	// Una entrada fuera de horario que no debe contarse en el límite de 10.
	entries[0].Product.Schedules = []dtos.ScheduleResponse{{StartTime: "00:00:00", EndTime: "23:59:59", Enabled: false}}

	m.repo.On("GetTrendingByCoords", mock.Anything, coords).Return(entries, nil)
	m.bcv.On("GetRate", mock.Anything).Return(&bcvmodels.ExchangeRate{Rate: 40}, nil)

	got, err := svc.GetTrendingByCoords(context.Background(), coords)

	require.NoError(t, err)
	assert.Len(t, got, 10)
	assert.Equal(t, 400.0, got[0].Product.Bs)
	assert.Equal(t, 80.0, got[0].Shop.DeliveryFeeBs)
}

func TestSearchCategoriesByShopID_DelegatesToRepo(t *testing.T) {
	svc, m := newProductAppFixture(t)
	shopID := uuid.New()
	m.repo.On("SearchCategoriesByShopID", mock.Anything, shopID).Return(models.Categories{{ShopID: shopID, Name: "Bebidas"}}, nil)

	got, err := svc.SearchCategoriesByShopID(context.Background(), shopID)

	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "Bebidas", got[0].Name)
}

func TestSearchCategorySummariesByShopID_DelegatesToRepo(t *testing.T) {
	svc, m := newProductAppFixture(t)
	shopID := uuid.New()
	m.repo.On("SearchCategorySummariesByShopID", mock.Anything, shopID, "adiciones").
		Return(models.Categories{{ShopID: shopID, Name: "Salsas"}}, nil)

	got, err := svc.SearchCategorySummariesByShopID(context.Background(), shopID, "adiciones")

	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "Salsas", got[0].Name)
}

func TestSearchMacroCategories_InvalidShopType(t *testing.T) {
	svc, _ := newProductAppFixture(t)

	_, err := svc.SearchMacroCategories(context.Background(), "NotAType")

	assert.ErrorIs(t, err, errcustom.ErrInvalidShopType)
}

func TestSearchMacroCategories_Success(t *testing.T) {
	svc, m := newProductAppFixture(t)
	m.repo.On("SearchMacroCategories", mock.Anything, "Restaurant").
		Return(models.MacroCategories{{Name: "Comidas", Slug: "comidas", ShopType: "Restaurant"}}, nil)

	got, err := svc.SearchMacroCategories(context.Background(), "Restaurant")

	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "Comidas", got[0].Name)
}
