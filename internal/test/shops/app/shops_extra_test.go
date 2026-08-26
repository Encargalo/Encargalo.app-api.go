package app_test

import (
	"context"
	"testing"
	"time"

	bcvmodels "Encargalo.app-api.go/internal/bcv/domain/models"
	mocksbcv "Encargalo.app-api.go/internal/mocks/bcv/domain/ports"
	mocksshops "Encargalo.app-api.go/internal/mocks/shops/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/shops/app"
	"Encargalo.app-api.go/internal/shops/domain/dtos"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGetShopsBy_AppliesBsRate(t *testing.T) {
	repo := mocksshops.NewShopsRepo(t)
	bcv := mocksbcv.NewBcvApp(t)
	shopsApp := app.NewShopsApp(repo, nil, bcv)

	criteria := dtos.SearchShopsByID{}
	coords := dtos.Coords{Latitude: 10.48, Longitude: -66.9}
	repo.On("GetShopsBy", context.Background(), criteria, coords).
		Return(dtos.ShopsResponse{{DeliveryFee: 2}}, nil)
	bcv.On("GetRate", mock.Anything).Return(&bcvmodels.ExchangeRate{Rate: 40}, nil)

	result, err := shopsApp.GetShopsBy(context.Background(), criteria, coords)

	require.NoError(t, err)
	require.Len(t, result, 1)
	require.Equal(t, 80.0, result[0].DeliveryFeeBs)
}

func TestGetShopsBy_RepoError(t *testing.T) {
	repo := mocksshops.NewShopsRepo(t)
	shopsApp := app.NewShopsApp(repo, nil, nil)

	criteria := dtos.SearchShopsByID{}
	coords := dtos.Coords{}
	repo.On("GetShopsBy", context.Background(), criteria, coords).
		Return(nil, errcustom.ErrUnexpectedError)

	_, err := shopsApp.GetShopsBy(context.Background(), criteria, coords)

	require.ErrorIs(t, err, errcustom.ErrUnexpectedError)
}

func TestGetShopByID_DelegatesToRepo(t *testing.T) {
	repo := mocksshops.NewShopsRepo(t)
	shopsApp := app.NewShopsApp(repo, nil, nil)

	shopID := uuid.New()
	repo.On("GetShopByID", context.Background(), shopID).Return(dtos.ShopResponse{ID: shopID}, nil)

	resp, err := shopsApp.GetShopByID(context.Background(), shopID)

	require.NoError(t, err)
	require.Equal(t, shopID, resp.ID)
}

func TestGetLaunchPromoEndsAt_DelegatesToRepo(t *testing.T) {
	repo := mocksshops.NewShopsRepo(t)
	shopsApp := app.NewShopsApp(repo, nil, nil)

	shopID := uuid.New()
	endsAt := time.Now().Add(48 * time.Hour)
	repo.On("GetLaunchPromoEndsAt", context.Background(), shopID).Return(&endsAt, nil)

	resp, err := shopsApp.GetLaunchPromoEndsAt(context.Background(), shopID)

	require.NoError(t, err)
	require.Equal(t, &endsAt, resp)
}

// SearchShops filtra los productos de un shop por su horario de disponibilidad
// (filterMatchedItemsBySchedule -> toShopsWindows); ver bizclock.IsAvailableAt.
func TestSearchShops_FiltersMatchedItemsOutsideSchedule(t *testing.T) {
	repo := mocksshops.NewShopsRepo(t)
	bcv := mocksbcv.NewBcvApp(t)
	shopsApp := app.NewShopsApp(repo, nil, bcv)

	shopID := uuid.New()
	query := dtos.SearchShopsQuery{Q: "arroz"}
	repo.On("SearchShopCandidates", context.Background(), query).
		Return([]dtos.SearchShopCandidate{{ShopResponse: dtos.ShopResponse{ID: shopID}}}, nil)
	repo.On("FindMatchedProducts", context.Background(), []uuid.UUID{shopID}, "arroz").
		Return(map[uuid.UUID][]dtos.MatchedProductItem{
			// Enabled: false garantiza que la ventana nunca aplique, sin
			// depender de la hora en que corra el test.
			shopID: {
				{Name: "fuera de horario", Schedules: []dtos.MatchedSchedule{
					{StartTime: "00:00", EndTime: "23:59", Enabled: false},
				}},
			},
		}, nil)
	bcv.On("GetRate", mock.Anything).Return(&bcvmodels.ExchangeRate{Rate: 40}, nil)

	result, err := shopsApp.SearchShops(context.Background(), query)

	require.NoError(t, err)
	require.Len(t, result, 1)
	require.Empty(t, result[0].MatchedProducts)
}

func TestSearchShops_RepoErrorOnCandidates(t *testing.T) {
	repo := mocksshops.NewShopsRepo(t)
	shopsApp := app.NewShopsApp(repo, nil, nil)

	query := dtos.SearchShopsQuery{}
	repo.On("SearchShopCandidates", context.Background(), query).
		Return(nil, errcustom.ErrUnexpectedError)

	_, err := shopsApp.SearchShops(context.Background(), query)

	require.ErrorIs(t, err, errcustom.ErrUnexpectedError)
}
