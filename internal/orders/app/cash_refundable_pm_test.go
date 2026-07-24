package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCashRefundablePMUSD_MixedBalanceAndPagoMovil(t *testing.T) {
	// Orden 57: 10 balance + 47 PM → solo 47 cash-refundable.
	got := cashRefundablePMUSD(57, 10, 47, true)
	assert.Equal(t, 47.0, got)
}

func TestCashRefundablePMUSD_IgnoresPMThatIncludesBalance(t *testing.T) {
	// Bug: details guardó PM = total (57) aunque hubo balance 10.
	got := cashRefundablePMUSD(57, 10, 57, true)
	assert.Equal(t, 47.0, got)
}

func TestCashRefundablePMUSD_DerivesWhenPMMissing(t *testing.T) {
	got := cashRefundablePMUSD(57, 10, 0, true)
	assert.Equal(t, 47.0, got)
}

func TestCashRefundablePMUSD_HundredPercentPagoMovil(t *testing.T) {
	got := cashRefundablePMUSD(57, 0, 57, true)
	assert.Equal(t, 57.0, got)
}

func TestCashRefundablePMUSD_LegacyNoDetails(t *testing.T) {
	got := cashRefundablePMUSD(57, 0, 0, false)
	assert.Equal(t, 57.0, got)
}

func TestCashRefundablePMUSD_HundredPercentBalance(t *testing.T) {
	got := cashRefundablePMUSD(10, 10, 0, true)
	assert.Equal(t, 0.0, got)
}
