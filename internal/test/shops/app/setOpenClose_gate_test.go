package app_test

import (
	"context"
	"testing"

	mocksbcv "Encargalo.app-api.go/internal/mocks/bcv/domain/ports"
	mocksshops "Encargalo.app-api.go/internal/mocks/shops/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/shops/app"
	shopsmodels "Encargalo.app-api.go/internal/shops/domain/models/shops"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestSetOpenClose_RejectsOpenWhenDraft(t *testing.T) {
	repo := mocksshops.NewShopsRepo(t)
	broadcaster := mocksshops.NewShopStatusBroadcaster(t)
	bcv := mocksbcv.NewBcvApp(t)
	shopsApp := app.NewShopsApp(repo, broadcaster, bcv)

	shopID := uuid.New()
	repo.On("GetShopStatus", context.Background(), shopID).Return(string(shopsmodels.ShopStatusDraft), nil)

	err := shopsApp.SetOpenClose(context.Background(), &shopID, true)

	require.ErrorIs(t, err, errcustom.ErrShopNotActive)
	repo.AssertNotCalled(t, "SetOpenClose", mock.Anything, mock.Anything, mock.Anything)
}

func TestSetOpenClose_AllowsOpenWhenActive(t *testing.T) {
	repo := mocksshops.NewShopsRepo(t)
	broadcaster := mocksshops.NewShopStatusBroadcaster(t)
	bcv := mocksbcv.NewBcvApp(t)
	shopsApp := app.NewShopsApp(repo, broadcaster, bcv)

	shopID := uuid.New()
	repo.On("GetShopStatus", context.Background(), shopID).Return(string(shopsmodels.ShopStatusActive), nil)
	repo.On("SetOpenClose", context.Background(), &shopID, true).Return(nil)
	broadcaster.On("BroadcastShopStatus", shopID, true).Return(nil)

	err := shopsApp.SetOpenClose(context.Background(), &shopID, true)

	require.NoError(t, err)
}

func TestSetOpenClose_ClosingSkipsStatusCheck(t *testing.T) {
	repo := mocksshops.NewShopsRepo(t)
	broadcaster := mocksshops.NewShopStatusBroadcaster(t)
	bcv := mocksbcv.NewBcvApp(t)
	shopsApp := app.NewShopsApp(repo, broadcaster, bcv)

	shopID := uuid.New()
	repo.On("SetOpenClose", context.Background(), &shopID, false).Return(nil)
	repo.On("TryCloseOpenBatch", context.Background(), shopID).Return(nil)
	broadcaster.On("BroadcastShopStatus", shopID, false).Return(nil)

	err := shopsApp.SetOpenClose(context.Background(), &shopID, false)

	require.NoError(t, err)
	repo.AssertNotCalled(t, "GetShopStatus", mock.Anything, mock.Anything)
}
