package app_test

import (
	"context"
	"strings"
	"testing"

	mockstracking "Encargalo.app-api.go/internal/mocks/tracking/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/tracking/app"
	"Encargalo.app-api.go/internal/tracking/domain/dtos"
	"Encargalo.app-api.go/internal/tracking/domain/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestTrack_InjectsSessionIntoParams(t *testing.T) {
	repo := mockstracking.NewTrackingRepo(t)
	trackingApp := app.New(repo)

	userID := uuid.New()
	sessionID := uuid.New()
	eventID := uuid.New()

	repo.On("Insert", mock.Anything, mock.MatchedBy(func(e models.Event) bool {
		return e.ID == eventID &&
			e.Type == "pixel" &&
			e.Params["name"] == "shop_view" &&
			e.Params["user_id"] == userID.String() &&
			e.Params["session_id"] == sessionID.String() &&
			e.Params["shop_id"] == "shop-1"
	})).Return(nil)

	err := trackingApp.Track(context.Background(), userID, sessionID, dtos.TrackEvent{
		ID:   eventID,
		Type: "pixel",
		Params: map[string]any{
			"name":    "shop_view",
			"shop_id": "shop-1",
			"user_id": "spoofed",
		},
	})

	require.NoError(t, err)
}

func TestTrack_NilParamsBecomesObject(t *testing.T) {
	repo := mockstracking.NewTrackingRepo(t)
	trackingApp := app.New(repo)

	userID := uuid.New()
	sessionID := uuid.New()

	repo.On("Insert", mock.Anything, mock.MatchedBy(func(e models.Event) bool {
		return e.Params["user_id"] == userID.String() && e.Params["session_id"] == sessionID.String()
	})).Return(nil)

	err := trackingApp.Track(context.Background(), userID, sessionID, dtos.TrackEvent{
		ID:   uuid.New(),
		Type: "cta",
	})

	require.NoError(t, err)
}

func TestTrack_InvalidType(t *testing.T) {
	repo := mockstracking.NewTrackingRepo(t)
	trackingApp := app.New(repo)

	err := trackingApp.Track(context.Background(), uuid.New(), uuid.New(), dtos.TrackEvent{
		ID:   uuid.New(),
		Type: "PIXEL",
	})

	require.ErrorIs(t, err, errcustom.ErrInvalidTrackingType)
	repo.AssertNotCalled(t, "Insert", mock.Anything, mock.Anything)
}

func TestTrack_HyphenTypeRejected(t *testing.T) {
	repo := mockstracking.NewTrackingRepo(t)
	trackingApp := app.New(repo)

	err := trackingApp.Track(context.Background(), uuid.New(), uuid.New(), dtos.TrackEvent{
		ID:   uuid.New(),
		Type: "shop-view",
	})

	require.ErrorIs(t, err, errcustom.ErrInvalidTrackingType)
}

func TestTrack_ParamsTooLarge(t *testing.T) {
	repo := mockstracking.NewTrackingRepo(t)
	trackingApp := app.New(repo)

	err := trackingApp.Track(context.Background(), uuid.New(), uuid.New(), dtos.TrackEvent{
		ID:     uuid.New(),
		Type:   "pixel",
		Params: map[string]any{"payload": strings.Repeat("x", 5000)},
	})

	require.ErrorIs(t, err, errcustom.ErrTrackingParamsTooLarge)
	repo.AssertNotCalled(t, "Insert", mock.Anything, mock.Anything)
}
