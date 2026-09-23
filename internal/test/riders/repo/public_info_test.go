package repo_test

import (
	"context"
	"testing"

	mockriders "Encargalo.app-api.go/internal/mocks/riders/domain/ports"
	ridersModels "Encargalo.app-api.go/internal/riders/domain/models"
	"Encargalo.app-api.go/internal/riders/repo"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestRiderPublicInfoLookup_CopiesPublicFieldsOnly(t *testing.T) {
	profiles := mockriders.NewRiderProfileRepo(t)
	lookup := repo.NewRiderPublicInfoLookup(profiles)
	riderID := uuid.New()
	photo := "https://cdn/photo.jpg"
	kyc := "https://cdn/cedula.jpg"
	plate := "AB123CD"
	color := "rojo"

	profiles.On("GetAccountBasics", mock.Anything, riderID).
		Return("Ada", "Lovelace", "04120000000", "ada@example.com", nil)
	profiles.On("GetByUserID", mock.Anything, riderID).Return(&ridersModels.RiderProfile{
		ProfilePhotoURL: &photo,
		VehiclePlate:    &plate,
		VehicleColor:    &color,
		IDPhotoURL:      &kyc,
		LicensePhotoURL: &kyc,
	}, nil)

	info, err := lookup.GetByRiderID(context.Background(), riderID)

	require.NoError(t, err)
	require.Equal(t, "Ada", info.Name)
	require.Equal(t, "Lovelace", info.SurName)
	require.Equal(t, "04120000000", info.Phone)
	require.Equal(t, &photo, info.ProfilePhotoURL)
	require.Equal(t, &plate, info.VehiclePlate)
	require.Equal(t, &color, info.VehicleColor)
}

func TestRiderPublicInfoLookup_ProfileMissingStillReturnsAccount(t *testing.T) {
	profiles := mockriders.NewRiderProfileRepo(t)
	lookup := repo.NewRiderPublicInfoLookup(profiles)
	riderID := uuid.New()

	profiles.On("GetAccountBasics", mock.Anything, riderID).Return("Ada", "", "", "", nil)
	profiles.On("GetByUserID", mock.Anything, riderID).Return(nil, errcustom.ErrNotFound)

	info, err := lookup.GetByRiderID(context.Background(), riderID)

	require.NoError(t, err)
	require.Equal(t, "Ada", info.Name)
	require.Nil(t, info.VehiclePlate)
}
