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

func TestRegisterItemDiscount_Success(t *testing.T) {
	svc, m := newProductAppFixture(t)
	itemID := uuid.New()
	shopID := uuid.New()
	m.repo.On("SearchItemByID", mock.Anything, itemID).Return(&models.Item{ID: itemID, ShopID: shopID, Price: 100}, nil)
	m.repo.On("RegisterItemDiscount", mock.Anything, mock.MatchedBy(func(d *models.ItemDiscount) bool {
		return d.ItemID == itemID && d.ShopID == shopID && d.DiscountType == models.DiscountTypePercentage
	})).Return(nil)

	discount := dtos.RegisterItemDiscount{ItemID: itemID, ShopID: shopID, DiscountType: models.DiscountTypePercentage, Value: 20}
	err := svc.RegisterItemDiscount(context.Background(), discount)

	require.NoError(t, err)
}

func TestRegisterItemDiscount_ItemNotFound(t *testing.T) {
	svc, m := newProductAppFixture(t)
	itemID := uuid.New()
	m.repo.On("SearchItemByID", mock.Anything, itemID).Return(nil, nil)

	err := svc.RegisterItemDiscount(context.Background(), dtos.RegisterItemDiscount{ItemID: itemID})

	assert.ErrorIs(t, err, errcustom.ErrNotFound)
}

func TestRegisterItemDiscount_Unauthorized(t *testing.T) {
	svc, m := newProductAppFixture(t)
	itemID := uuid.New()
	m.repo.On("SearchItemByID", mock.Anything, itemID).Return(&models.Item{ID: itemID, ShopID: uuid.New()}, nil)

	err := svc.RegisterItemDiscount(context.Background(), dtos.RegisterItemDiscount{ItemID: itemID, ShopID: uuid.New()})

	assert.ErrorIs(t, err, errcustom.ErrUnautorized)
}

func TestRegisterItemDiscount_InvalidPercentage(t *testing.T) {
	svc, m := newProductAppFixture(t)
	itemID := uuid.New()
	shopID := uuid.New()
	m.repo.On("SearchItemByID", mock.Anything, itemID).Return(&models.Item{ID: itemID, ShopID: shopID, Price: 100}, nil)

	discount := dtos.RegisterItemDiscount{ItemID: itemID, ShopID: shopID, DiscountType: models.DiscountTypePercentage, Value: 150}
	err := svc.RegisterItemDiscount(context.Background(), discount)

	assert.ErrorIs(t, err, errcustom.ErrInvalidDiscountPercent)
}

func TestRegisterItemDiscount_InvalidFixedExceedsPrice(t *testing.T) {
	svc, m := newProductAppFixture(t)
	itemID := uuid.New()
	shopID := uuid.New()
	m.repo.On("SearchItemByID", mock.Anything, itemID).Return(&models.Item{ID: itemID, ShopID: shopID, Price: 100}, nil)

	discount := dtos.RegisterItemDiscount{ItemID: itemID, ShopID: shopID, DiscountType: models.DiscountTypeFixed, Value: 100}
	err := svc.RegisterItemDiscount(context.Background(), discount)

	assert.ErrorIs(t, err, errcustom.ErrInvalidDiscountFixed)
}

func TestRegisterItemDiscount_InvalidDiscountType(t *testing.T) {
	svc, m := newProductAppFixture(t)
	itemID := uuid.New()
	shopID := uuid.New()
	m.repo.On("SearchItemByID", mock.Anything, itemID).Return(&models.Item{ID: itemID, ShopID: shopID, Price: 100}, nil)

	discount := dtos.RegisterItemDiscount{ItemID: itemID, ShopID: shopID, DiscountType: "unknown", Value: 10}
	err := svc.RegisterItemDiscount(context.Background(), discount)

	assert.ErrorIs(t, err, errcustom.ErrInvalidDiscountValue)
}

func TestUpdateItemDiscount_NotFound(t *testing.T) {
	svc, m := newProductAppFixture(t)
	discountID := uuid.New()
	m.repo.On("SearchItemDiscountByID", mock.Anything, discountID).Return(nil, nil)

	err := svc.UpdateItemDiscount(context.Background(), dtos.UpdateItemDiscount{}, discountID, uuid.New())

	assert.ErrorIs(t, err, errcustom.ErrNotFound)
}

func TestUpdateItemDiscount_Unauthorized(t *testing.T) {
	svc, m := newProductAppFixture(t)
	discountID := uuid.New()
	m.repo.On("SearchItemDiscountByID", mock.Anything, discountID).Return(&models.ItemDiscount{ID: discountID, ShopID: uuid.New()}, nil)

	err := svc.UpdateItemDiscount(context.Background(), dtos.UpdateItemDiscount{}, discountID, uuid.New())

	assert.ErrorIs(t, err, errcustom.ErrUnautorized)
}

func TestUpdateItemDiscount_Success(t *testing.T) {
	svc, m := newProductAppFixture(t)
	discountID := uuid.New()
	shopID := uuid.New()
	itemID := uuid.New()
	existing := &models.ItemDiscount{ID: discountID, ShopID: shopID, ItemID: itemID}
	m.repo.On("SearchItemDiscountByID", mock.Anything, discountID).Return(existing, nil)
	m.repo.On("SearchItemByID", mock.Anything, itemID).Return(&models.Item{ID: itemID, ShopID: shopID, Price: 100}, nil)
	m.repo.On("UpdateItemDiscount", mock.Anything, mock.MatchedBy(func(u *models.UpdateItemDiscount) bool {
		return u.ID == discountID && u.Value == 30
	}), shopID, itemID).Return(nil)

	discount := dtos.UpdateItemDiscount{DiscountType: models.DiscountTypePercentage, Value: 30, IsActive: true}
	err := svc.UpdateItemDiscount(context.Background(), discount, discountID, shopID)

	require.NoError(t, err)
}

func TestDeleteItemDiscount_NotFound(t *testing.T) {
	svc, m := newProductAppFixture(t)
	discountID := uuid.New()
	m.repo.On("SearchItemDiscountByID", mock.Anything, discountID).Return(nil, nil)

	err := svc.DeleteItemDiscount(context.Background(), discountID, uuid.New())

	assert.ErrorIs(t, err, errcustom.ErrNotFound)
}

func TestDeleteItemDiscount_Unauthorized(t *testing.T) {
	svc, m := newProductAppFixture(t)
	discountID := uuid.New()
	m.repo.On("SearchItemDiscountByID", mock.Anything, discountID).Return(&models.ItemDiscount{ID: discountID, ShopID: uuid.New()}, nil)

	err := svc.DeleteItemDiscount(context.Background(), discountID, uuid.New())

	assert.ErrorIs(t, err, errcustom.ErrUnautorized)
}

func TestDeleteItemDiscount_Success(t *testing.T) {
	svc, m := newProductAppFixture(t)
	discountID := uuid.New()
	shopID := uuid.New()
	existing := &models.ItemDiscount{ID: discountID, ShopID: shopID}
	m.repo.On("SearchItemDiscountByID", mock.Anything, discountID).Return(existing, nil)
	m.repo.On("DeleteItemDiscount", mock.Anything, existing).Return(nil)

	err := svc.DeleteItemDiscount(context.Background(), discountID, shopID)

	require.NoError(t, err)
}

func TestDeleteItemDiscountByItemID_ItemNotFound(t *testing.T) {
	svc, m := newProductAppFixture(t)
	itemID := uuid.New()
	m.repo.On("SearchItemByID", mock.Anything, itemID).Return(nil, nil)

	err := svc.DeleteItemDiscountByItemID(context.Background(), itemID, uuid.New())

	assert.ErrorIs(t, err, errcustom.ErrNotFound)
}

func TestDeleteItemDiscountByItemID_Unauthorized(t *testing.T) {
	svc, m := newProductAppFixture(t)
	itemID := uuid.New()
	m.repo.On("SearchItemByID", mock.Anything, itemID).Return(&models.Item{ID: itemID, ShopID: uuid.New()}, nil)

	err := svc.DeleteItemDiscountByItemID(context.Background(), itemID, uuid.New())

	assert.ErrorIs(t, err, errcustom.ErrUnautorized)
}

func TestDeleteItemDiscountByItemID_NoActiveDiscount(t *testing.T) {
	svc, m := newProductAppFixture(t)
	itemID := uuid.New()
	shopID := uuid.New()
	m.repo.On("SearchItemByID", mock.Anything, itemID).Return(&models.Item{ID: itemID, ShopID: shopID}, nil)
	m.repo.On("SearchActiveItemDiscountByItemID", mock.Anything, itemID).Return(nil, nil)

	err := svc.DeleteItemDiscountByItemID(context.Background(), itemID, shopID)

	assert.ErrorIs(t, err, errcustom.ErrNotFound)
}

func TestDeleteItemDiscountByItemID_Success(t *testing.T) {
	svc, m := newProductAppFixture(t)
	itemID := uuid.New()
	shopID := uuid.New()
	discount := &models.ItemDiscount{ID: uuid.New(), ItemID: itemID, ShopID: shopID}
	m.repo.On("SearchItemByID", mock.Anything, itemID).Return(&models.Item{ID: itemID, ShopID: shopID}, nil)
	m.repo.On("SearchActiveItemDiscountByItemID", mock.Anything, itemID).Return(discount, nil)
	m.repo.On("DeleteItemDiscount", mock.Anything, discount).Return(nil)

	err := svc.DeleteItemDiscountByItemID(context.Background(), itemID, shopID)

	require.NoError(t, err)
}
