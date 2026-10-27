package app_test

import (
	"context"
	"testing"

	bcvmodels "Encargalo.app-api.go/internal/bcv/domain/models"
	mocksbcv "Encargalo.app-api.go/internal/mocks/bcv/domain/ports"
	mocksshops "Encargalo.app-api.go/internal/mocks/shops/domain/ports"
	"Encargalo.app-api.go/internal/shops/app"
	"Encargalo.app-api.go/internal/shops/domain/dtos"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGetAllShops_PropagatesTypeFilter(t *testing.T) {
	repo := mocksshops.NewShopsRepo(t)
	bcv := mocksbcv.NewBcvApp(t)
	shopsApp := app.NewShopsApp(repo, nil, bcv, nil)

	query := dtos.ListShopsQuery{
		Latitude:  4.678034,
		Longitude: -74.0496399,
		Type:      "Supermarket",
	}

	repo.On("GetAllShops", context.Background(), query).Return(dtos.ShopsResponse{
		{Type: "Supermarket", DeliveryFee: 1.5},
	}, nil)
	bcv.On("GetRate", mock.Anything).Return(&bcvmodels.ExchangeRate{Rate: 40}, nil)

	result, err := shopsApp.GetAllShops(context.Background(), query)
	require.NoError(t, err)
	require.Len(t, result, 1)
	require.Equal(t, "Supermarket", result[0].Type)
	repo.AssertExpectations(t)
}

func TestSearchShops_PropagatesTypeFilter(t *testing.T) {
	repo := mocksshops.NewShopsRepo(t)
	bcv := mocksbcv.NewBcvApp(t)
	shopsApp := app.NewShopsApp(repo, nil, bcv, nil)

	query := dtos.SearchShopsQuery{
		Q:         "arroz",
		Latitude:  4.678034,
		Longitude: -74.0496399,
		Type:      "Restaurant",
	}

	repo.On("SearchShopCandidates", context.Background(), query).Return([]dtos.SearchShopCandidate{}, nil)
	bcv.On("GetRate", mock.Anything).Return(&bcvmodels.ExchangeRate{Rate: 40}, nil).Maybe()

	result, err := shopsApp.SearchShops(context.Background(), query)
	require.NoError(t, err)
	require.Empty(t, result)
	repo.AssertExpectations(t)
}
