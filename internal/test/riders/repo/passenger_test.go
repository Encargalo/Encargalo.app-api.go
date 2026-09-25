package repo_test

import (
	"context"
	"testing"

	mockriders "Encargalo.app-api.go/internal/mocks/riders/domain/ports"
	"Encargalo.app-api.go/internal/riders/repo"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestPassengerLookup_ReturnsNameAndPhoneOnly(t *testing.T) {
	profiles := mockriders.NewRiderProfileRepo(t)
	lookup := repo.NewPassengerLookup(profiles)
	customerID := uuid.New()

	profiles.On("GetAccountBasics", mock.Anything, customerID).
		Return("Ana", "Pérez", "04141234567", "ana@example.com", nil)

	contact, err := lookup.GetByCustomerID(context.Background(), customerID)

	require.NoError(t, err)
	require.Equal(t, "Ana", contact.Name)
	require.Equal(t, "04141234567", contact.Phone)
}
