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
		{1234, 1230},
		{1235, 1240},
		{0, 0},
		{4, 0},
		{5, 10},
		{36.4521 * 12.99, 470},
	}

	for _, tc := range cases {
		assert.Equal(t, tc.expected, money.RoundBs(tc.input))
	}
}
