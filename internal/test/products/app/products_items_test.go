package app_test

import (
	"context"
	"testing"

	"Encargalo.app-api.go/internal/products/domain/dtos"
	"Encargalo.app-api.go/internal/products/domain/models"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const testImageB64 = "data:image/png;base64,aGVsbG8=" // "hello" en base64, basta para exercitar el hash/decode real.

func TestRegisterItem_Success(t *testing.T) {
	svc, m := newProductAppFixture(t)
	categoryID := uuid.New()
	shopID := uuid.New()
	flavorID := uuid.New()
	additionID := uuid.New()

	item := dtos.RegisterItem{
		ShopID:       shopID,
		CategoryID:   categoryID,
		Name:         "Pizza",
		Price:        100,
		Image:        testImageB64,
		Description:  "desc",
		FlavorIDs:    []uuid.UUID{flavorID},
		AdditionIDs:  []uuid.UUID{additionID},
		MinSelection: nil,
		MaxSelection: nil,
	}

	m.upload.On("UploadImage", mock.Anything, testImageB64).
		Return(&uploader.UploadResult{PublicID: "encargalo/pizza", SecureURL: "https://cdn/pizza.png"}, nil)
	category := &models.Category{
		ID: categoryID, ShopID: shopID,
		Details: models.CategoryDetails{
			SelectorType: models.SelectorSingleSelect,
			Flavors:      models.FlavorEntries{{ID: flavorID, Name: "Picante", Available: models.BoolPtr(true)}},
		},
	}
	m.repo.On("SearchCategoryByID", mock.Anything, categoryID).Return(category, nil)
	m.repo.On("SearchAdditionItemsByIDs", mock.Anything, shopID, []uuid.UUID{additionID}).
		Return(models.Items{{ID: additionID, Name: "Extra queso", Price: 10}}, nil)
	m.repo.On("RegisterItem", mock.Anything, mock.MatchedBy(func(it *models.Item) bool {
		return it.ShopID == shopID && it.CategoryID == categoryID && it.HasFlavors &&
			len(it.Details.Additions) == 1 && it.Details.Additions[0].Name == "Extra queso"
	})).Return(nil)

	err := svc.RegisterItem(context.Background(), item)

	require.NoError(t, err)
}

func TestRegisterItem_UploadImageFails(t *testing.T) {
	svc, m := newProductAppFixture(t)
	item := dtos.RegisterItem{ShopID: uuid.New(), CategoryID: uuid.New(), Image: testImageB64}

	m.upload.On("UploadImage", mock.Anything, testImageB64).Return(nil, assert.AnError)

	err := svc.RegisterItem(context.Background(), item)

	assert.ErrorIs(t, err, errcustom.ErrUploadingImage)
}

func TestRegisterItem_InvalidFlavors(t *testing.T) {
	svc, m := newProductAppFixture(t)
	categoryID := uuid.New()
	shopID := uuid.New()
	unknownFlavorID := uuid.New()

	item := dtos.RegisterItem{
		ShopID: shopID, CategoryID: categoryID, Image: testImageB64,
		FlavorIDs: []uuid.UUID{unknownFlavorID},
	}

	m.upload.On("UploadImage", mock.Anything, testImageB64).
		Return(&uploader.UploadResult{PublicID: "p", SecureURL: "https://cdn/p.png"}, nil)
	category := &models.Category{
		ID: categoryID, ShopID: shopID,
		Details: models.CategoryDetails{SelectorType: models.SelectorSingleSelect, Flavors: models.FlavorEntries{}},
	}
	m.repo.On("SearchCategoryByID", mock.Anything, categoryID).Return(category, nil)

	err := svc.RegisterItem(context.Background(), item)

	assert.ErrorIs(t, err, errcustom.ErrInvalidFlavors)
}

func TestUpdateItem_NotFound(t *testing.T) {
	svc, m := newProductAppFixture(t)
	itemID := uuid.New()
	m.repo.On("SearchItemByID", mock.Anything, itemID).Return(nil, nil)

	err := svc.UpdateItem(context.Background(), dtos.UpdateItem{}, itemID, uuid.New())

	assert.ErrorIs(t, err, errcustom.ErrNotFound)
}

func TestUpdateItem_Unauthorized(t *testing.T) {
	svc, m := newProductAppFixture(t)
	itemID := uuid.New()
	m.repo.On("SearchItemByID", mock.Anything, itemID).Return(&models.Item{ID: itemID, ShopID: uuid.New()}, nil)

	err := svc.UpdateItem(context.Background(), dtos.UpdateItem{}, itemID, uuid.New())

	assert.ErrorIs(t, err, errcustom.ErrUnautorized)
}

// testImageB64Hash es sha256("hello"), el payload decodificado de testImageB64.
const testImageB64Hash = "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"

// TestUpdateItem_SameImageHashSkipsUpload cubre la rama que evita re-subir a
// Cloudinary cuando el hash de la imagen enviada coincide con la persistida.
// No se configura expectativa en m.upload: si UpdateItem la invocara, mockery
// haría panic por una llamada no esperada.
func TestUpdateItem_SameImageHashSkipsUpload(t *testing.T) {
	svc, m := newProductAppFixture(t)
	itemID := uuid.New()
	shopID := uuid.New()

	existing := &models.Item{
		ID: itemID, ShopID: shopID, CategoryID: uuid.New(),
		Image: "https://cdn/old.png", ImageHash: testImageB64Hash,
		AvailabilityStatus: models.AvailabilityActive,
	}
	m.repo.On("SearchItemByID", mock.Anything, itemID).Return(existing, nil)
	m.repo.On("UpdateItem", mock.Anything, mock.MatchedBy(func(u *models.UpdateItem) bool {
		return u.Image == "https://cdn/old.png"
	}), shopID).Return(nil)
	m.repo.On("IsAdditionItem", mock.Anything, itemID).Return(false, nil)

	err := svc.UpdateItem(context.Background(), dtos.UpdateItem{Name: "Nueva", Price: 50, Image: testImageB64}, itemID, shopID)

	require.NoError(t, err)
}

func TestUpdateItem_ImageChangedUploadsNewImage(t *testing.T) {
	svc, m := newProductAppFixture(t)
	itemID := uuid.New()
	shopID := uuid.New()

	existing := &models.Item{
		ID: itemID, ShopID: shopID, CategoryID: uuid.New(),
		Image: "https://cdn/old.png", ImageHash: "different-hash",
		AvailabilityStatus: models.AvailabilityActive,
	}
	m.repo.On("SearchItemByID", mock.Anything, itemID).Return(existing, nil)
	m.upload.On("UploadImage", mock.Anything, testImageB64).
		Return(&uploader.UploadResult{PublicID: "p2", SecureURL: "https://cdn/new.png"}, nil)
	m.repo.On("UpdateItem", mock.Anything, mock.MatchedBy(func(u *models.UpdateItem) bool {
		return u.Name == "Nueva"
	}), shopID).Return(nil)
	m.repo.On("IsAdditionItem", mock.Anything, itemID).Return(false, nil)

	err := svc.UpdateItem(context.Background(), dtos.UpdateItem{Name: "Nueva", Price: 50, Image: testImageB64}, itemID, shopID)

	require.NoError(t, err)
}

func TestUpdateItem_SyncsAdditionSnapshotWhenItemIsAnAddition(t *testing.T) {
	svc, m := newProductAppFixture(t)
	itemID := uuid.New()
	shopID := uuid.New()

	existing := &models.Item{
		ID: itemID, ShopID: shopID, CategoryID: uuid.New(),
		AvailabilityStatus: models.AvailabilityActive,
	}
	m.repo.On("SearchItemByID", mock.Anything, itemID).Return(existing, nil)
	m.repo.On("UpdateItem", mock.Anything, mock.Anything, shopID).Return(nil)
	m.repo.On("IsAdditionItem", mock.Anything, itemID).Return(true, nil)
	m.repo.On("SyncAdditionSnapshots", mock.Anything, shopID, mock.MatchedBy(func(e models.AdditionEntry) bool {
		return e.ID == itemID && e.Name == "Nueva"
	})).Return(nil)

	err := svc.UpdateItem(context.Background(), dtos.UpdateItem{Name: "Nueva", Price: 50}, itemID, shopID)

	require.NoError(t, err)
}
