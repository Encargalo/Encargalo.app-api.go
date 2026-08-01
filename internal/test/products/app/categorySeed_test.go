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

func TestSeedDefaultCategories_CreatesOneCategoryPerMicro(t *testing.T) {
	repo := mocksproducts.NewProductsRepo(t)
	productsApp := app.NewProductsApp(repo, nil, nil, nil)

	shopID := uuid.New()
	macroID := uuid.New()
	micros := models.MicroCategories{
		{ID: uuid.New(), MacroCategoryID: macroID, Name: "Licuados y Jugos", Slug: "licuados-y-jugos"},
		{ID: uuid.New(), MacroCategoryID: macroID, Name: "Malteadas", Slug: "malteadas"},
	}
	repo.On("SearchMicroCategories", context.Background(), "Restaurant").Return(micros, nil)
	repo.On("RegisterCategory", context.Background(), mock.MatchedBy(func(c *models.Category) bool {
		return c.ShopID == shopID && c.MacroCategoryID != nil && *c.MacroCategoryID == macroID &&
			(c.Name == "Licuados y Jugos" || c.Name == "Malteadas")
	})).Return(nil).Twice()

	err := productsApp.SeedDefaultCategories(context.Background(), shopID, "Restaurant")

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestSeedDefaultCategories_NoMicrosForType(t *testing.T) {
	repo := mocksproducts.NewProductsRepo(t)
	productsApp := app.NewProductsApp(repo, nil, nil, nil)

	shopID := uuid.New()
	repo.On("SearchMicroCategories", context.Background(), "Turbo").Return(models.MicroCategories{}, nil)

	err := productsApp.SeedDefaultCategories(context.Background(), shopID, "Turbo")

	require.NoError(t, err)
	repo.AssertNotCalled(t, "RegisterCategory", mock.Anything, mock.Anything)
}
