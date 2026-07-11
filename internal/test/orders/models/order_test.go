package models_test

import (
	"testing"

	ordersModels "Encargalo.app-api.go/internal/orders/domain/models"
	productsModels "Encargalo.app-api.go/internal/products/domain/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestOrder_SetPrices_UsesShopPrice(t *testing.T) {
	itemID := uuid.New()
	order := &ordersModels.Order{
		DeliveryFee: 2,
		ItemsOrders: []ordersModels.ItemsOrder{
			{ItemID: itemID, Amount: 2},
		},
	}

	items := productsModels.Items{
		{ID: itemID, Price: 10},
	}

	order.SetPrices(items)

	assert.Equal(t, 10, order.ItemsOrders[0].UnitPrice)
	assert.Equal(t, 20, order.ItemsOrders[0].TotalPrice)
	assert.Equal(t, float64(22), order.TotalPrice)
}

func TestOrder_ApplyLaunchPromo_Active(t *testing.T) {
	order := &ordersModels.Order{
		DeliveryFee: 2,
		TotalPrice:  22,
	}

	order.ApplyLaunchPromo(true)

	assert.True(t, order.IsLaunchPromo)
	assert.Equal(t, 10.0, order.PromoSubsidyAmount)
	assert.Equal(t, 12.0, order.TotalPrice)
}

func TestOrder_ApplyLaunchPromo_Inactive(t *testing.T) {
	order := &ordersModels.Order{
		DeliveryFee: 2,
		TotalPrice:  22,
	}

	order.ApplyLaunchPromo(false)

	assert.False(t, order.IsLaunchPromo)
	assert.Equal(t, 0.0, order.PromoSubsidyAmount)
	assert.Equal(t, 22.0, order.TotalPrice)
}

func TestOrder_ApplyZeroCommission(t *testing.T) {
	order := &ordersModels.Order{}

	order.ApplyZeroCommission(true)
	assert.True(t, order.IsZeroCommission)

	order.ApplyZeroCommission(false)
	assert.False(t, order.IsZeroCommission)
}
