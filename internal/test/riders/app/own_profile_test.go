package app_test

import (
	"context"
	"testing"

	mockuploadimage "Encargalo.app-api.go/internal/mocks/pkg/uploadImage"
	mocksriders "Encargalo.app-api.go/internal/mocks/riders/domain/ports"
	"Encargalo.app-api.go/internal/riders/app"
	"Encargalo.app-api.go/internal/riders/domain/models"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func strPtr(s string) *string { return &s }

func TestGetOwnProfile_ReturnsDriverAndVehicleOnly(t *testing.T) {
	repo := mocksriders.NewRiderProfileRepo(t)
	svc := app.NewRiderProfileApp(repo, mockuploadimage.NewUploadImage(t))

	userID := uuid.New()
	repo.On("GetAccountBasics", mock.Anything, userID).Return("Ada", "Lovelace", "0414", "a@b.c", nil)
	repo.On("GetByUserID", mock.Anything, userID).Return(&models.RiderProfile{
		UserID:          userID,
		ProfilePhotoURL: strPtr("https://img/profile.jpg"),
		VehicleType:     strPtr("moto"),
		VehiclePlate:    strPtr("ABCD123"),
		VehicleBrand:    strPtr("Yamaha"),
		VehicleModel:    strPtr("FZ"),
		VehicleColor:    strPtr("Amarillo"),
		IDPhotoURL:      strPtr("https://img/id.jpg"),
		Notes:           strPtr("interno"),
	}, nil)

	resp, err := svc.GetOwnProfile(context.Background(), userID)

	require.NoError(t, err)
	assert.Equal(t, userID.String(), resp.Driver.UserID)
	assert.Equal(t, "Ada", resp.Driver.Name)
	assert.Equal(t, "Lovelace", resp.Driver.SurName)
	assert.Equal(t, "https://img/profile.jpg", *resp.Driver.ProfilePhotoURL)
	assert.Equal(t, "moto", *resp.Vehicle.Type)
	assert.Equal(t, "ABCD123", *resp.Vehicle.Plate)
	assert.Equal(t, "Yamaha", *resp.Vehicle.Brand)
	assert.Equal(t, "FZ", *resp.Vehicle.Model)
	assert.Equal(t, "Amarillo", *resp.Vehicle.Color)
}

func TestGetOwnProfile_NoProfileYetReturnsEmptyVehicle(t *testing.T) {
	repo := mocksriders.NewRiderProfileRepo(t)
	svc := app.NewRiderProfileApp(repo, mockuploadimage.NewUploadImage(t))

	userID := uuid.New()
	repo.On("GetAccountBasics", mock.Anything, userID).Return("Ada", "Lovelace", "0414", "a@b.c", nil)
	repo.On("GetByUserID", mock.Anything, userID).Return(nil, errcustom.ErrNotFound)

	resp, err := svc.GetOwnProfile(context.Background(), userID)

	require.NoError(t, err)
	assert.Equal(t, "Ada", resp.Driver.Name)
	assert.Nil(t, resp.Driver.ProfilePhotoURL)
	assert.Empty(t, resp.Vehicle)
}

func TestGetOwnProfile_BikeWithoutPlate(t *testing.T) {
	repo := mocksriders.NewRiderProfileRepo(t)
	svc := app.NewRiderProfileApp(repo, mockuploadimage.NewUploadImage(t))

	userID := uuid.New()
	repo.On("GetAccountBasics", mock.Anything, userID).Return("Ada", "Lovelace", "0414", "a@b.c", nil)
	repo.On("GetByUserID", mock.Anything, userID).Return(&models.RiderProfile{UserID: userID, VehicleType: strPtr(models.VehicleTypeBike)}, nil)

	resp, err := svc.GetOwnProfile(context.Background(), userID)

	require.NoError(t, err)
	assert.Equal(t, models.VehicleTypeBike, *resp.Vehicle.Type)
	assert.Nil(t, resp.Vehicle.Plate)
}

func TestGetOwnProfile_AccountErrorPropagates(t *testing.T) {
	repo := mocksriders.NewRiderProfileRepo(t)
	svc := app.NewRiderProfileApp(repo, mockuploadimage.NewUploadImage(t))

	userID := uuid.New()
	repo.On("GetAccountBasics", mock.Anything, userID).Return("", "", "", "", errcustom.ErrNotFound)

	_, err := svc.GetOwnProfile(context.Background(), userID)

	assert.ErrorIs(t, err, errcustom.ErrNotFound)
}

func TestGetOwnProfile_ProfileRepoErrorPropagates(t *testing.T) {
	repo := mocksriders.NewRiderProfileRepo(t)
	svc := app.NewRiderProfileApp(repo, mockuploadimage.NewUploadImage(t))

	userID := uuid.New()
	repo.On("GetAccountBasics", mock.Anything, userID).Return("Ada", "Lovelace", "0414", "a@b.c", nil)
	repo.On("GetByUserID", mock.Anything, userID).Return(nil, errcustom.ErrUnexpectedError)

	_, err := svc.GetOwnProfile(context.Background(), userID)

	assert.ErrorIs(t, err, errcustom.ErrUnexpectedError)
}
