package app_test

import (
	"context"
	"testing"

	mocksproducts "Encargalo.app-api.go/internal/mocks/products/domain/ports"
	"Encargalo.app-api.go/internal/products/app"
	"Encargalo.app-api.go/internal/products/domain/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestSeedDefaultCategories_CreatesOneCategoryPerMacro(t *testing.T) {
	repo := mocksproducts.NewProductsRepo(t)
	productsApp := app.NewProductsApp(repo, nil, nil, nil)

	shopID := uuid.New()
	macros := models.MacroCategories{
		{ID: uuid.New(), Name: "Hamburguesas", Slug: "hamburguesas", ShopType: "Restaurant"},
		{ID: uuid.New(), Name: "Pizzas", Slug: "pizzas", ShopType: "Restaurant"},
	}
	repo.On("SearchMacroCategories", context.Background(), "Restaurant").Return(macros, nil)
	repo.On("RegisterCategory", context.Background(), mock.MatchedBy(func(c *models.Category) bool {
		return c.ShopID == shopID && c.MacroCategoryID != nil && (c.Name == "Hamburguesas" || c.Name == "Pizzas")
	})).Return(nil).Twice()

	err := productsApp.SeedDefaultCategories(context.Background(), shopID, "Restaurant")

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestSeedDefaultCategories_NoMacrosForType(t *testing.T) {
	repo := mocksproducts.NewProductsRepo(t)
	productsApp := app.NewProductsApp(repo, nil, nil, nil)

	shopID := uuid.New()
	repo.On("SearchMacroCategories", context.Background(), "Turbo").Return(models.MacroCategories{}, nil)

	err := productsApp.SeedDefaultCategories(context.Background(), shopID, "Turbo")

	require.NoError(t, err)
	repo.AssertNotCalled(t, "RegisterCategory", mock.Anything, mock.Anything)
}
