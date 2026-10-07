package models_test

import (
	"strconv"
	"testing"

	"Encargalo.app-api.go/internal/users/domain/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestBuildActivateAccount_SixDigitCodeInRange(t *testing.T) {
	customerID := uuid.New()

	for range 500 {
		var acc models.ActivateAccount
		require.NoError(t, acc.BuildActivateAccount(customerID))

		require.Equal(t, customerID, acc.CustomerID)
		require.Len(t, acc.ActivationCode, 6)
		n, err := strconv.Atoi(acc.ActivationCode)
		require.NoError(t, err)
		require.GreaterOrEqual(t, n, 100_000)
		require.LessOrEqual(t, n, 999_999)
	}
}

// Con math/rand sembrado por segundo, todos los códigos del mismo segundo salían iguales.
func TestBuildActivateAccount_CodesInSameInstantDiffer(t *testing.T) {
	seen := make(map[string]struct{})
	for range 200 {
		var acc models.ActivateAccount
		require.NoError(t, acc.BuildActivateAccount(uuid.New()))
		seen[acc.ActivationCode] = struct{}{}
	}

	require.Greater(t, len(seen), 190)
}
