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

func TestRegisterCategory_SuccessWithFlavorsAndSchedules(t *testing.T) {
	svc, m := newProductAppFixture(t)
	shopID := uuid.New()

	category := dtos.RegisterCategory{
		ShopID:          shopID,
		Name:            "Bebidas",
		MacroCategoryID: uuid.New(),
		SelectorType:    models.SelectorSingleSelect,
		Flavors:         []dtos.FlavorInput{{Name: "Fresa"}, {Name: "Mora"}},
		Schedules:       []dtos.ScheduleInput{{StartTime: "08:00:00", EndTime: "12:00:00"}},
	}

	m.repo.On("RegisterCategory", mock.Anything, mock.MatchedBy(func(c *models.Category) bool {
		return c.ShopID == shopID && c.Name == "Bebidas" && len(c.Details.Flavors) == 2
	})).Return(nil)
	m.repo.On("ReplaceCategorySchedules", mock.Anything, mock.Anything, shopID, mock.MatchedBy(func(s []models.Schedule) bool {
		return len(s) == 1 && s[0].StartTime == "08:00:00"
	})).Return(nil)

	err := svc.RegisterCategory(context.Background(), category)

	require.NoError(t, err)
}

func TestRegisterCategory_NoSchedulesSkipsReplaceCall(t *testing.T) {
	svc, m := newProductAppFixture(t)
	category := dtos.RegisterCategory{ShopID: uuid.New(), Name: "Postres", MacroCategoryID: uuid.New()}

	m.repo.On("RegisterCategory", mock.Anything, mock.Anything).Return(nil)

	err := svc.RegisterCategory(context.Background(), category)

	require.NoError(t, err)
	m.repo.AssertNotCalled(t, "ReplaceCategorySchedules", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestRegisterCategory_InvalidSelectorType(t *testing.T) {
	svc, _ := newProductAppFixture(t)
	category := dtos.RegisterCategory{ShopID: uuid.New(), Name: "X", SelectorType: "not_a_type"}

	err := svc.RegisterCategory(context.Background(), category)

	assert.ErrorIs(t, err, errcustom.ErrInvalidSelection)
}

func TestRegisterCategory_FlavorsRequireSelectorSupport(t *testing.T) {
	svc, _ := newProductAppFixture(t)
	category := dtos.RegisterCategory{
		ShopID: uuid.New(), Name: "X", SelectorType: models.SelectorNone,
		Flavors: []dtos.FlavorInput{{Name: "Fresa"}},
	}

	err := svc.RegisterCategory(context.Background(), category)

	assert.ErrorIs(t, err, errcustom.ErrCategoryWithoutFlavors)
}

func TestRegisterCategory_DuplicateFlavorName(t *testing.T) {
	svc, _ := newProductAppFixture(t)
	category := dtos.RegisterCategory{
		ShopID: uuid.New(), Name: "X", SelectorType: models.SelectorSingleSelect,
		Flavors: []dtos.FlavorInput{{Name: "Fresa"}, {Name: "Fresa"}},
	}

	err := svc.RegisterCategory(context.Background(), category)

	assert.ErrorIs(t, err, errcustom.ErrInvalidSelection)
}

func TestUpdateCategory_NotFound(t *testing.T) {
	svc, m := newProductAppFixture(t)
	categoryID := uuid.New()
	m.repo.On("SearchCategoryByID", mock.Anything, categoryID).Return(nil, nil)

	err := svc.UpdateCategory(context.Background(), dtos.UpdateCategory{}, categoryID, uuid.New())

	assert.ErrorIs(t, err, errcustom.ErrNotFound)
}

func TestUpdateCategory_Unauthorized(t *testing.T) {
	svc, m := newProductAppFixture(t)
	categoryID := uuid.New()
	m.repo.On("SearchCategoryByID", mock.Anything, categoryID).Return(&models.Category{ID: categoryID, ShopID: uuid.New()}, nil)

	err := svc.UpdateCategory(context.Background(), dtos.UpdateCategory{}, categoryID, uuid.New())

	assert.ErrorIs(t, err, errcustom.ErrUnautorized)
}

func TestUpdateCategory_CannotDisableSelectorWithExistingFlavors(t *testing.T) {
	svc, m := newProductAppFixture(t)
	categoryID := uuid.New()
	shopID := uuid.New()
	existing := &models.Category{
		ID: categoryID, ShopID: shopID,
		Details: models.CategoryDetails{
			SelectorType: models.SelectorSingleSelect,
			Flavors:      models.FlavorEntries{{ID: uuid.New(), Name: "Fresa"}},
		},
	}
	m.repo.On("SearchCategoryByID", mock.Anything, categoryID).Return(existing, nil)

	none := models.SelectorNone
	err := svc.UpdateCategory(context.Background(), dtos.UpdateCategory{SelectorType: &none}, categoryID, shopID)

	assert.ErrorIs(t, err, errcustom.ErrCategoryHasFlavors)
}

func TestUpdateCategory_SuccessReplacesSchedules(t *testing.T) {
	svc, m := newProductAppFixture(t)
	categoryID := uuid.New()
	shopID := uuid.New()
	existing := &models.Category{ID: categoryID, ShopID: shopID, Details: models.DefaultCategoryDetails()}
	m.repo.On("SearchCategoryByID", mock.Anything, categoryID).Return(existing, nil)
	m.repo.On("UpdateCategory", mock.Anything, mock.MatchedBy(func(u *models.UpdateCategory) bool {
		return u.Name == "Nueva"
	}), shopID).Return(nil)
	schedules := []dtos.ScheduleInput{{StartTime: "09:00:00", EndTime: "10:00:00"}}
	m.repo.On("ReplaceCategorySchedules", mock.Anything, categoryID, shopID, mock.Anything).Return(nil)

	err := svc.UpdateCategory(context.Background(), dtos.UpdateCategory{Name: "Nueva", MacroCategoryID: uuid.New(), Schedules: &schedules}, categoryID, shopID)

	require.NoError(t, err)
}
