package models_test

import (
	"testing"

	"Encargalo.app-api.go/internal/shops/domain/models"
	"github.com/stretchr/testify/assert"
)

func TestComputeShopSaleBreakdown_RegularSale(t *testing.T) {
	breakdown := models.ComputeShopSaleBreakdown(52, 2, 0, false, false)

	assert.Equal(t, 50.0, breakdown.Base)
	assert.Equal(t, models.ShopShare, breakdown.Share)
	assert.Equal(t, 5.0, breakdown.Commission)
	assert.Equal(t, 45.0, breakdown.Net)
}

func TestComputeShopSaleBreakdown_ZeroCommission(t *testing.T) {
	breakdown := models.ComputeShopSaleBreakdown(52, 2, 0, false, true)

	assert.Equal(t, 50.0, breakdown.Base)
	assert.Equal(t, models.ZeroCommissionShare, breakdown.Share)
	assert.Equal(t, 0.0, breakdown.Commission)
	assert.Equal(t, 50.0, breakdown.Net)
}

func TestComputeShopSaleBreakdown_LaunchPromo(t *testing.T) {
	breakdown := models.ComputeShopSaleBreakdown(27, 2, 25, true, true)

	assert.Equal(t, 50.0, breakdown.Base)
	assert.Equal(t, 50.0, breakdown.Net)
	assert.Equal(t, 0.0, breakdown.Commission)
}
