package app_test

import (
	"context"
	"testing"

	mockuploadimage "Encargalo.app-api.go/internal/mocks/pkg/uploadImage"
	mocksriders "Encargalo.app-api.go/internal/mocks/riders/domain/ports"
	"Encargalo.app-api.go/internal/riders/app"
	"Encargalo.app-api.go/internal/riders/domain/dtos"
	"Encargalo.app-api.go/internal/riders/domain/models"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestUpsertDeviceToken_SavesTrimmedTokenForRider(t *testing.T) {
	repo := mocksriders.NewRiderProfileRepo(t)
	svc := app.NewRiderProfileApp(repo, mockuploadimage.NewUploadImage(t), nil)
	userID := uuid.New()

	repo.On("UpsertDeviceToken", mock.Anything, mock.MatchedBy(func(token models.RiderDeviceToken) bool {
		return token.RiderID == userID && token.FCMToken == "abc123" && token.Platform == models.DevicePlatformAndroid
	})).Return(nil).Once()

	err := svc.UpsertDeviceToken(context.Background(), userID, dtos.UpsertDeviceTokenRequest{
		FCMToken: "  abc123  ",
		Platform: models.DevicePlatformAndroid,
	})

	require.NoError(t, err)
}

func TestUpsertDeviceToken_PropagatesRepoError(t *testing.T) {
	repo := mocksriders.NewRiderProfileRepo(t)
	svc := app.NewRiderProfileApp(repo, mockuploadimage.NewUploadImage(t), nil)
	userID := uuid.New()

	repo.On("UpsertDeviceToken", mock.Anything, mock.Anything).Return(errcustom.ErrUnexpectedError).Once()

	err := svc.UpsertDeviceToken(context.Background(), userID, dtos.UpsertDeviceTokenRequest{
		FCMToken: "abc123",
		Platform: models.DevicePlatformIOS,
	})

	assert.ErrorIs(t, err, errcustom.ErrUnexpectedError)
}
