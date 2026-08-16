package app_test

import (
	"testing"

	"Encargalo.app-api.go/internal/orders/app"
	ordermodels "Encargalo.app-api.go/internal/orders/domain/models"
	shopmodels "Encargalo.app-api.go/internal/shops/domain/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildOrderBroadcast_ShopRegularSale(t *testing.T) {
	order := sampleOrder(52, 2, false, false, 0)

	broadcast := app.BuildOrderBroadcast("order_update", order, true)

	require.NotNil(t, broadcast.SaleAmount)
	require.NotNil(t, broadcast.Commission)
	require.NotNil(t, broadcast.NetAmount)
	assert.Equal(t, "order_update", broadcast.Type)
	assert.Equal(t, 50.0, *broadcast.SaleAmount)
	assert.Equal(t, 5.0, *broadcast.Commission)
	assert.Equal(t, 45.0, *broadcast.NetAmount)
	assert.Equal(t, 50.0, broadcast.Order.TotalPrice)
	assert.Equal(t, 0.0, broadcast.Order.DeliveryFee)
	assert.Nil(t, broadcast.Order.PickupCode)
	assert.Nil(t, broadcast.Order.DeliveryCode)
}

func TestBuildOrderBroadcast_ShopZeroCommission(t *testing.T) {
	order := sampleOrder(52, 2, false, true, 0)

	broadcast := app.BuildOrderBroadcast("order_update", order, true)

	require.NotNil(t, broadcast.NetAmount)
	assert.Equal(t, 50.0, *broadcast.SaleAmount)
	assert.Equal(t, 0.0, *broadcast.Commission)
	assert.Equal(t, 50.0, *broadcast.NetAmount)
}

func TestBuildOrderBroadcast_ShopLaunchPromo(t *testing.T) {
	order := sampleOrder(27, 2, true, true, 25)

	broadcast := app.BuildOrderBroadcast("order_update", order, true)

	require.NotNil(t, broadcast.NetAmount)
	assert.Equal(t, 50.0, *broadcast.SaleAmount)
	assert.Equal(t, 0.0, *broadcast.Commission)
	assert.Equal(t, 50.0, *broadcast.NetAmount)
}

func TestBuildOrderBroadcast_CustomerOmitsCommissionFields(t *testing.T) {
	pickup := "1234"
	order := sampleOrder(52, 2, false, false, 0)
	order.PickupCode = &pickup

	broadcast := app.BuildOrderBroadcast("order_update", order, false)

	assert.Nil(t, broadcast.SaleAmount)
	assert.Nil(t, broadcast.Commission)
	assert.Nil(t, broadcast.NetAmount)
	assert.Equal(t, 52.0, broadcast.Order.TotalPrice)
	assert.Equal(t, 2.0, broadcast.Order.DeliveryFee)
	assert.Nil(t, broadcast.Order.PickupCode)
}

func TestBuildOrderBroadcast_MatchesShopFinanceBreakdown(t *testing.T) {
	cases := []struct {
		name               string
		totalPrice         float64
		deliveryFee        float64
		isLaunchPromo      bool
		isZeroCommission   bool
		promoSubsidyAmount float64
	}{
		{"regular sale", 52, 2, false, false, 0},
		{"zero commission", 52, 2, false, true, 0},
		{"launch promo", 27, 2, true, true, 25},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			order := sampleOrder(tc.totalPrice, tc.deliveryFee, tc.isLaunchPromo, tc.isZeroCommission, tc.promoSubsidyAmount)
			expected := shopmodels.ComputeShopSaleBreakdown(
				tc.totalPrice,
				tc.deliveryFee,
				tc.promoSubsidyAmount,
				tc.isLaunchPromo,
				tc.isZeroCommission,
			)

			broadcast := app.BuildOrderBroadcast("order_update", order, true)

			assert.Equal(t, expected.Base, *broadcast.SaleAmount)
			assert.Equal(t, expected.Commission, *broadcast.Commission)
			assert.Equal(t, expected.Net, *broadcast.NetAmount)
		})
	}
}

func sampleOrder(totalPrice, deliveryFee float64, isLaunchPromo, isZeroCommission bool, promoSubsidy float64) ordermodels.Order {
	pickup := "1234"
	delivery := "5678"
	return ordermodels.Order{
		ID:                 uuid.New(),
		ShopID:             uuid.New(),
		CustomerID:         uuid.New(),
		TotalPrice:         totalPrice,
		DeliveryFee:        deliveryFee,
		IsLaunchPromo:      isLaunchPromo,
		IsZeroCommission:   isZeroCommission,
		PromoSubsidyAmount: promoSubsidy,
		PickupCode:         &pickup,
		DeliveryCode:       &delivery,
	}
}
