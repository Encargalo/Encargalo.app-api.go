package app_test

import (
	"context"
	"testing"

	mocksshops "Encargalo.app-api.go/internal/mocks/shops/domain/ports"
	"Encargalo.app-api.go/internal/shops/app"
	"Encargalo.app-api.go/internal/shops/domain/dtos"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Punto de referencia del cliente para todos los casos de este archivo.
const (
	customerLat = 10.4806
	customerLon = -66.9036
)

func TestSelectNearestOpenTurboNode_ChoosesNearest(t *testing.T) {
	near := dtos.ShopResponse{ID: uuid.New(), Opened: true, Latitude: 10.4820, Longitude: customerLon, Score: 1}
	far := dtos.ShopResponse{ID: uuid.New(), Opened: true, Latitude: 10.4900, Longitude: customerLon, Score: 10}

	best, _, found := app.SelectNearestOpenTurboNode(dtos.ShopsResponse{far, near}, customerLat, customerLon)

	require.True(t, found)
	require.Equal(t, near.ID, best.ID)
}

func TestSelectNearestOpenTurboNode_IgnoresClosedShops(t *testing.T) {
	closedButNear := dtos.ShopResponse{ID: uuid.New(), Opened: false, Latitude: 10.4810, Longitude: customerLon}
	openButFarther := dtos.ShopResponse{ID: uuid.New(), Opened: true, Latitude: 10.4900, Longitude: customerLon}

	best, _, found := app.SelectNearestOpenTurboNode(dtos.ShopsResponse{closedButNear, openButFarther}, customerLat, customerLon)

	require.True(t, found)
	require.Equal(t, openButFarther.ID, best.ID)
}

func TestSelectNearestOpenTurboNode_RespectsRadius(t *testing.T) {
	tooFar := dtos.ShopResponse{ID: uuid.New(), Opened: true, Latitude: 10.6000, Longitude: customerLon}

	_, _, found := app.SelectNearestOpenTurboNode(dtos.ShopsResponse{tooFar}, customerLat, customerLon)

	require.False(t, found)
}

func TestSelectNearestOpenTurboNode_NoCandidates(t *testing.T) {
	_, _, found := app.SelectNearestOpenTurboNode(dtos.ShopsResponse{}, customerLat, customerLon)

	require.False(t, found)
}

func TestGetTurboEntry_AssignsNearestAndResolvesMetrics(t *testing.T) {
	repo := mocksshops.NewShopsRepo(t)
	shopsApp := app.NewShopsApp(repo, nil, nil)

	near := dtos.ShopResponse{ID: uuid.New(), Name: "Turbo Cerca", Opened: true, Latitude: 10.4820, Longitude: customerLon}
	far := dtos.ShopResponse{ID: uuid.New(), Name: "Turbo Lejos", Opened: true, Latitude: 10.4900, Longitude: customerLon}
	coords := dtos.Coords{Latitude: customerLat, Longitude: customerLon}

	repo.On("ListTurboCandidates", context.Background()).Return(dtos.ShopsResponse{far, near}, nil)
	repo.On("ResolveTurboShop", context.Background(), near.ID, coords).Return(dtos.ShopResponse{
		ID: near.ID, Name: near.Name, Type: "Turbo", Opened: true, Time: 8, DeliveryFee: 1.2,
	}, nil)

	entry, err := shopsApp.GetTurboEntry(context.Background(), coords)

	require.NoError(t, err)
	require.True(t, entry.Available)
	require.NotNil(t, entry.Shop)
	require.Equal(t, near.ID, entry.Shop.ID)
	require.NotNil(t, entry.EtaMinutes)
	require.Equal(t, 8, *entry.EtaMinutes)
	repo.AssertExpectations(t)
}

func TestGetTurboEntry_NoCoverageReturnsUnavailableWithoutResolvingMetrics(t *testing.T) {
	repo := mocksshops.NewShopsRepo(t)
	shopsApp := app.NewShopsApp(repo, nil, nil)

	coords := dtos.Coords{Latitude: customerLat, Longitude: customerLon}
	repo.On("ListTurboCandidates", context.Background()).Return(dtos.ShopsResponse{}, nil)

	entry, err := shopsApp.GetTurboEntry(context.Background(), coords)

	require.NoError(t, err)
	require.False(t, entry.Available)
	require.Nil(t, entry.Shop)
	repo.AssertExpectations(t)
	repo.AssertNotCalled(t, "ResolveTurboShop", mock.Anything, mock.Anything, mock.Anything)
}
