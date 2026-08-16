package money_test

import (
	"testing"

	"Encargalo.app-api.go/internal/pkg/money"
	"github.com/stretchr/testify/require"
)

func TestNativeDeltaFromStaffBs_RiderDebit(t *testing.T) {
	delta, rate := money.NativeDeltaFromStaffBs(5200, money.CurrencyVES, 52, true)
	require.NotNil(t, rate)
	require.Equal(t, 52.0, *rate)
	require.Equal(t, -5200.0, delta)
}

func TestNativeDeltaFromStaffBs_ShopDebit(t *testing.T) {
	delta, rate := money.NativeDeltaFromStaffBs(5200, money.CurrencyUSD, 52, true)
	require.NotNil(t, rate)
	require.Equal(t, -100.0, delta)
}

func TestNativeDeltaFromStaffBs_ShopCredit(t *testing.T) {
	delta, _ := money.NativeDeltaFromStaffBs(2600, money.CurrencyUSD, 52, false)
	require.Equal(t, 50.0, delta)
}
