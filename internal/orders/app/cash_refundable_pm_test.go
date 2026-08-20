package app

import (
	"testing"

	"Encargalo.app-api.go/internal/orders/domain/models"
	"github.com/stretchr/testify/assert"
)

func TestCashRefundablePMBs_MixedBalanceAndPagoMovil(t *testing.T) {
	// Orden 2280 Bs: 400 balance + 1880 PM → solo 1880 cash-refundable.
	got := cashRefundablePMBs(2280, 400, 1880, true)
	assert.Equal(t, 1880.0, got)
}

func TestCashRefundablePMBs_IgnoresPMThatIncludesBalance(t *testing.T) {
	// Bug: details guardó PM = total (2280) aunque hubo balance 400.
	got := cashRefundablePMBs(2280, 400, 2280, true)
	assert.Equal(t, 1880.0, got)
}

func TestCashRefundablePMBs_DerivesWhenPMMissing(t *testing.T) {
	got := cashRefundablePMBs(2280, 400, 0, true)
	assert.Equal(t, 1880.0, got)
}

func TestCashRefundablePMBs_HundredPercentPagoMovil(t *testing.T) {
	got := cashRefundablePMBs(2280, 0, 2280, true)
	assert.Equal(t, 2280.0, got)
}

func TestCashRefundablePMBs_LegacyNoDetails(t *testing.T) {
	got := cashRefundablePMBs(2280, 0, 0, false)
	assert.Equal(t, 2280.0, got)
}

func TestCashRefundablePMBs_HundredPercentBalance(t *testing.T) {
	got := cashRefundablePMBs(400, 400, 0, true)
	assert.Equal(t, 0.0, got)
}

func TestPaymentSplitFromDetails_LegacyUSDFallback(t *testing.T) {
	order := &models.Order{
		Details: []byte(`{"payment":{"balance_amount_bs":400,"pago_movil_amount_usd":47}}`),
	}
	balanceBs, pmBs, ok := paymentSplitFromDetails(order, 40)
	assert.True(t, ok)
	assert.Equal(t, 400.0, balanceBs)
	assert.Equal(t, 1880.0, pmBs)
}

func TestPaymentSplitFromDetails_PrefersBsOverLegacyUSD(t *testing.T) {
	order := &models.Order{
		Details: []byte(`{"payment":{"balance_amount_bs":400,"pago_movil_amount_bs":1880,"pago_movil_amount_usd":47}}`),
	}
	balanceBs, pmBs, ok := paymentSplitFromDetails(order, 40)
	assert.True(t, ok)
	assert.Equal(t, 400.0, balanceBs)
	assert.Equal(t, 1880.0, pmBs)
}
