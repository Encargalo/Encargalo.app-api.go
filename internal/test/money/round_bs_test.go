package money_test

import (
	"testing"

	"Encargalo.app-api.go/internal/pkg/money"
	"github.com/stretchr/testify/assert"
)

func TestRoundBs(t *testing.T) {
	cases := []struct {
		input    float64
		expected float64
	}{
		{500.215, 500.22},
		{500.214, 500.21},
		{0, 0},
		{1.999, 2.00},
		{36.4521 * 12.99, 473.51},
		{36.4521 * 0.5, 18.23},
	}

	for _, tc := range cases {
		assert.Equal(t, tc.expected, money.RoundBs(tc.input))
	}
}
