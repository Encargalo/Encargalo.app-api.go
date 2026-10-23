package models_test

import (
	"testing"

	ordersModels "Encargalo.app-api.go/internal/orders/domain/models"
	riderModels "Encargalo.app-api.go/internal/riders/domain/models"
	"github.com/stretchr/testify/assert"
)

func TestFirstOrderDeliverySubsidyUSD_CoversWholeFeeUnderCap(t *testing.T) {
	assert.Equal(t, 2.0, ordersModels.FirstOrderDeliverySubsidyUSD(2))
}

func TestFirstOrderDeliverySubsidyUSD_CappedAtThreeDollars(t *testing.T) {
	assert.Equal(t, 3.0, ordersModels.FirstOrderDeliverySubsidyUSD(5))
}

func TestApplyFirstOrderDeliverySubsidy_CustomerPaysOnlyOverCap(t *testing.T) {
	order := ordersModels.Order{DeliveryFee: 5}

	order.ApplyFirstOrderDeliverySubsidy(true)

	assert.Equal(t, 2.0, order.DeliveryFee)
	assert.Equal(t, 3.0, order.DeliverySubsidyUSD)
	assert.Equal(t, 5.0, order.RiderDeliveryFee())
}

func TestApplyFirstOrderDeliverySubsidy_NotEligibleKeepsFee(t *testing.T) {
	order := ordersModels.Order{DeliveryFee: 5}

	order.ApplyFirstOrderDeliverySubsidy(false)

	assert.Equal(t, 5.0, order.DeliveryFee)
	assert.Zero(t, order.DeliverySubsidyUSD)
}

// El rider ve su 90% sobre el envío completo aunque el cliente no lo pague.
func TestNewRiderOrder_FreeDeliveryShowsPayOnFullFee(t *testing.T) {
	order := ordersModels.Order{DeliveryFee: 0, DeliverySubsidyUSD: 2, TotalPrice: 20, MethodPayment: string(ordersModels.MethodEfectivo)}
	order.ApplyBsPrice(40)

	rider := riderModels.NewRiderOrder(order, false)

	assert.InDelta(t, 1.8, rider.DeliveryFee, 0.001)
	assert.InDelta(t, 72.0, rider.DeliveryFeeBs, 0.001)
	assert.NotNil(t, rider.CollectUSD)
	assert.Equal(t, 20.0, *rider.CollectUSD, "cobra solo lo que paga el cliente")
}
