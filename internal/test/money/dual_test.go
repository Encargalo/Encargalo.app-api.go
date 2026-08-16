package money_test

import (
	"testing"

	"Encargalo.app-api.go/internal/pkg/money"
	"github.com/stretchr/testify/assert"
)

func TestBalanceDual_VES(t *testing.T) {
	bs, usd := money.BalanceDual(148.5, money.CurrencyVES, 36.5)
	assert.Equal(t, 148.5, bs)
	assert.Equal(t, 4.07, usd)
}

func TestBalanceDual_USD(t *testing.T) {
	bs, usd := money.BalanceDual(15.0, money.CurrencyUSD, 36.5)
	assert.Equal(t, 547.5, bs)
	assert.Equal(t, 15.0, usd)
}

func TestTransactionDual_VESWithFrozenRate(t *testing.T) {
	rate := 40.0
	bs, usd := money.TransactionDual(80.0, money.CurrencyVES, &rate, 36.5)
	assert.Equal(t, 80.0, bs)
	assert.Equal(t, 2.0, usd)
}

func TestTransactionDual_USDWithSpotRate(t *testing.T) {
	bs, usd := money.TransactionDual(10.0, money.CurrencyUSD, nil, 36.5)
	assert.Equal(t, 365.0, bs)
	assert.Equal(t, 10.0, usd)
}
