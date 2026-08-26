package app_test

import (
	"context"
	"testing"

	"Encargalo.app-api.go/internal/products/domain/dtos"
	"Encargalo.app-api.go/internal/products/domain/models"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestRegisterFlavor_Success(t *testing.T) {
	svc, m := newProductAppFixture(t)
	categoryID := uuid.New()
	shopID := uuid.New()
	flavor := dtos.RegisterFlavor{ShopID: shopID, CategoryID: categoryID, Name: "Fresa"}

	category := &models.Category{
		ID: categoryID, ShopID: shopID,
		Details: models.CategoryDetails{SelectorType: models.SelectorSingleSelect},
	}
	m.repo.On("SearchCategoryByID", mock.Anything, categoryID).Return(category, nil)
	m.repo.On("RegisterFlavor", mock.Anything, mock.MatchedBy(func(f models.CatalogFlavor) bool {
		return f.ShopID == shopID && f.CategoryID == categoryID && f.Entry.Name == "Fresa"
	})).Return(nil)

	err := svc.RegisterFlavor(context.Background(), flavor)

	require.NoError(t, err)
}

func TestRegisterFlavor_CategoryNotFound(t *testing.T) {
	svc, m := newProductAppFixture(t)
	categoryID := uuid.New()
	m.repo.On("SearchCategoryByID", mock.Anything, categoryID).Return(nil, nil)

	err := svc.RegisterFlavor(context.Background(), dtos.RegisterFlavor{CategoryID: categoryID})

	assert.ErrorIs(t, err, errcustom.ErrNotFound)
}

func TestRegisterFlavor_Unauthorized(t *testing.T) {
	svc, m := newProductAppFixture(t)
	categoryID := uuid.New()
	m.repo.On("SearchCategoryByID", mock.Anything, categoryID).Return(&models.Category{ID: categoryID, ShopID: uuid.New()}, nil)

	err := svc.RegisterFlavor(context.Background(), dtos.RegisterFlavor{CategoryID: categoryID, ShopID: uuid.New()})

	assert.ErrorIs(t, err, errcustom.ErrUnautorized)
}

func TestRegisterFlavor_CategoryWithoutFlavorSupport(t *testing.T) {
	svc, m := newProductAppFixture(t)
	categoryID := uuid.New()
	shopID := uuid.New()
	category := &models.Category{ID: categoryID, ShopID: shopID, Details: models.CategoryDetails{SelectorType: models.SelectorNone}}
	m.repo.On("SearchCategoryByID", mock.Anything, categoryID).Return(category, nil)

	err := svc.RegisterFlavor(context.Background(), dtos.RegisterFlavor{CategoryID: categoryID, ShopID: shopID})

	assert.ErrorIs(t, err, errcustom.ErrCategoryWithoutFlavors)
}

func TestRegisterFlavor_DuplicateName(t *testing.T) {
	svc, m := newProductAppFixture(t)
	categoryID := uuid.New()
	shopID := uuid.New()
	category := &models.Category{
		ID: categoryID, ShopID: shopID,
		Details: models.CategoryDetails{
			SelectorType: models.SelectorSingleSelect,
			Flavors:      models.FlavorEntries{{ID: uuid.New(), Name: "Fresa"}},
		},
	}
	m.repo.On("SearchCategoryByID", mock.Anything, categoryID).Return(category, nil)

	err := svc.RegisterFlavor(context.Background(), dtos.RegisterFlavor{CategoryID: categoryID, ShopID: shopID, Name: "Fresa"})

	assert.ErrorIs(t, err, errcustom.ErrUnexpectedError)
}

func TestUpdateFlavor_NotFound(t *testing.T) {
	svc, m := newProductAppFixture(t)
	flavorID := uuid.New()
	m.repo.On("SearchFlavorByID", mock.Anything, flavorID).Return(nil, nil)

	err := svc.UpdateFlavor(context.Background(), dtos.UpdateFlavor{}, flavorID, uuid.New())

	assert.ErrorIs(t, err, errcustom.ErrNotFound)
}

func TestUpdateFlavor_Unauthorized(t *testing.T) {
	svc, m := newProductAppFixture(t)
	flavorID := uuid.New()
	m.repo.On("SearchFlavorByID", mock.Anything, flavorID).Return(&models.CatalogFlavor{ID: flavorID, ShopID: uuid.New()}, nil)

	err := svc.UpdateFlavor(context.Background(), dtos.UpdateFlavor{}, flavorID, uuid.New())

	assert.ErrorIs(t, err, errcustom.ErrUnautorized)
}

func TestUpdateFlavor_DuplicateName(t *testing.T) {
	svc, m := newProductAppFixture(t)
	flavorID := uuid.New()
	shopID := uuid.New()
	categoryID := uuid.New()
	flavorModel := &models.CatalogFlavor{ID: flavorID, ShopID: shopID, CategoryID: categoryID, Entry: models.FlavorEntry{ID: flavorID, Name: "Fresa"}}
	category := &models.Category{
		ID: categoryID, ShopID: shopID,
		Details: models.CategoryDetails{Flavors: models.FlavorEntries{{ID: uuid.New(), Name: "Mora"}}},
	}
	m.repo.On("SearchFlavorByID", mock.Anything, flavorID).Return(flavorModel, nil)
	m.repo.On("SearchCategoryByID", mock.Anything, categoryID).Return(category, nil)

	err := svc.UpdateFlavor(context.Background(), dtos.UpdateFlavor{Name: "Mora"}, flavorID, shopID)

	assert.ErrorIs(t, err, errcustom.ErrUnexpectedError)
}

func TestUpdateFlavor_Success(t *testing.T) {
	svc, m := newProductAppFixture(t)
	flavorID := uuid.New()
	shopID := uuid.New()
	categoryID := uuid.New()
	flavorModel := &models.CatalogFlavor{ID: flavorID, ShopID: shopID, CategoryID: categoryID, Entry: models.FlavorEntry{ID: flavorID, Name: "Fresa"}}
	category := &models.Category{ID: categoryID, ShopID: shopID, Details: models.CategoryDetails{Flavors: models.FlavorEntries{}}}
	m.repo.On("SearchFlavorByID", mock.Anything, flavorID).Return(flavorModel, nil)
	m.repo.On("SearchCategoryByID", mock.Anything, categoryID).Return(category, nil)
	available := false
	m.repo.On("UpdateFlavor", mock.Anything, flavorID, mock.MatchedBy(func(e models.FlavorEntry) bool {
		return e.Name == "Mora" && e.Available != nil && !*e.Available
	}), shopID, categoryID).Return(nil)

	err := svc.UpdateFlavor(context.Background(), dtos.UpdateFlavor{Name: "Mora", Available: &available}, flavorID, shopID)

	require.NoError(t, err)
}

func TestDeleteFlavor_NotFound(t *testing.T) {
	svc, m := newProductAppFixture(t)
	flavorID := uuid.New()
	m.repo.On("SearchFlavorByID", mock.Anything, flavorID).Return(nil, nil)

	err := svc.DeleteFlavor(context.Background(), flavorID, uuid.New())

	assert.ErrorIs(t, err, errcustom.ErrNotFound)
}

func TestDeleteFlavor_Unauthorized(t *testing.T) {
	svc, m := newProductAppFixture(t)
	flavorID := uuid.New()
	m.repo.On("SearchFlavorByID", mock.Anything, flavorID).Return(&models.CatalogFlavor{ID: flavorID, ShopID: uuid.New()}, nil)

	err := svc.DeleteFlavor(context.Background(), flavorID, uuid.New())

	assert.ErrorIs(t, err, errcustom.ErrUnautorized)
}

func TestDeleteFlavor_Success(t *testing.T) {
	svc, m := newProductAppFixture(t)
	flavorID := uuid.New()
	shopID := uuid.New()
	flavorModel := &models.CatalogFlavor{ID: flavorID, ShopID: shopID}
	m.repo.On("SearchFlavorByID", mock.Anything, flavorID).Return(flavorModel, nil)
	m.repo.On("DeleteFlavor", mock.Anything, *flavorModel).Return(nil)

	err := svc.DeleteFlavor(context.Background(), flavorID, shopID)

	require.NoError(t, err)
}
