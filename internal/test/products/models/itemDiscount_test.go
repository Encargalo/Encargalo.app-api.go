package models_test

import (
	"testing"
	"time"

	"Encargalo.app-api.go/internal/products/domain/models"
	"github.com/stretchr/testify/assert"
)

func TestItemDiscount_Apply_Percentage(t *testing.T) {
	d := &models.ItemDiscount{DiscountType: models.DiscountTypePercentage, Value: 20}
	assert.Equal(t, 80.0, d.Apply(100))
}

func TestItemDiscount_Apply_Fixed(t *testing.T) {
	d := &models.ItemDiscount{DiscountType: models.DiscountTypeFixed, Value: 30}
	assert.Equal(t, 70.0, d.Apply(100))
}

func TestItemDiscount_Apply_NeverNegative(t *testing.T) {
	d := &models.ItemDiscount{DiscountType: models.DiscountTypeFixed, Value: 500}
	assert.Equal(t, 0.0, d.Apply(100))
}

func TestItemDiscount_IsCurrentlyActive(t *testing.T) {
	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	cases := []struct {
		name     string
		discount models.ItemDiscount
		want     bool
	}{
		{"active no window", models.ItemDiscount{IsActive: true}, true},
		{"inactive toggle", models.ItemDiscount{IsActive: false}, false},
		{"soft deleted", models.ItemDiscount{IsActive: true, DeletedAt: &now}, false},
		{"not started yet", models.ItemDiscount{IsActive: true, StartsAt: &future}, false},
		{"already ended", models.ItemDiscount{IsActive: true, EndsAt: &past}, false},
		{"within window", models.ItemDiscount{IsActive: true, StartsAt: &past, EndsAt: &future}, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.discount.IsCurrentlyActive())
		})
	}
}

func TestItem_ShopPrice_NoDiscount(t *testing.T) {
	item := &models.Item{Price: 100}
	assert.Equal(t, 100.0, item.ShopPrice())
}

func TestItem_ShopPrice_WithActiveDiscount(t *testing.T) {
	item := &models.Item{
		Price:    100,
		Discount: &models.ItemDiscount{IsActive: true, DiscountType: models.DiscountTypePercentage, Value: 25},
	}
	assert.Equal(t, 75.0, item.ShopPrice())
}

func TestItem_ShopPrice_WithInactiveDiscount(t *testing.T) {
	item := &models.Item{
		Price:    100,
		Discount: &models.ItemDiscount{IsActive: false, DiscountType: models.DiscountTypePercentage, Value: 25},
	}
	assert.Equal(t, 100.0, item.ShopPrice())
}

func TestItemDiscount_Apply_DecimalPrice(t *testing.T) {
	d := &models.ItemDiscount{DiscountType: models.DiscountTypePercentage, Value: 10, IsActive: true}
	assert.Equal(t, 1.29, d.Apply(1.43))
}
