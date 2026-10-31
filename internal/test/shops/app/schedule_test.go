package app_test

import (
	"context"
	"testing"
	"time"

	bcvmodels "Encargalo.app-api.go/internal/bcv/domain/models"
	mocksbcv "Encargalo.app-api.go/internal/mocks/bcv/domain/ports"
	mocksshops "Encargalo.app-api.go/internal/mocks/shops/domain/ports"
	"Encargalo.app-api.go/internal/pkg/bizclock"
	"Encargalo.app-api.go/internal/shops/app"
	"Encargalo.app-api.go/internal/shops/domain/dtos"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func timeOfDay(t time.Time) string {
	return t.Format("15:04:05")
}

// windowAround construye una única ventana del día de hoy que cubre [now+from, now+to].
func windowAround(now time.Time, from, to time.Duration) dtos.ScheduleResponse {
	day := int16(now.Weekday())
	return dtos.ScheduleResponse{
		DayOfWeek: day,
		Open:      timeOfDay(now.Add(from)),
		Close:     timeOfDay(now.Add(to)),
		Enabled:   true,
	}
}

func TestGetAllShops_OpenFirstThenClosedSoon_HidesFarClosed(t *testing.T) {
	repo := mocksshops.NewShopsRepo(t)
	bcv := mocksbcv.NewBcvApp(t)
	shopsApp := app.NewShopsApp(repo, nil, bcv, nil)

	now := bizclock.Now()

	openShop := uuid.New()
	closedSoonShop := uuid.New()
	closedFarShop := uuid.New()

	query := dtos.ListShopsQuery{Latitude: 10.48, Longitude: -66.9}

	repo.On("GetAllShops", context.Background(), query).Return(dtos.ShopsResponse{
		{ID: openShop, Opened: true},
		{ID: closedSoonShop, Opened: false},
		{ID: closedFarShop, Opened: false},
	}, nil)
	bcv.On("GetRate", mock.Anything).Return(&bcvmodels.ExchangeRate{Rate: 40}, nil)

	repo.On("GetSchedulesByShopIDs", context.Background(), mock.Anything).Return(map[uuid.UUID][]dtos.ScheduleResponse{
		openShop:       {windowAround(now, -5*time.Minute, 1*time.Hour)},
		closedSoonShop: {windowAround(now, 15*time.Minute, 2*time.Hour)},
		closedFarShop:  {windowAround(now, 2*time.Hour, 3*time.Hour)},
	}, nil)

	result, err := shopsApp.GetAllShops(context.Background(), query)
	require.NoError(t, err)
	require.Len(t, result, 2)

	require.Equal(t, openShop, result[0].ID)
	require.True(t, result[0].Opened)
	require.NotNil(t, result[0].ClosesAt)
	require.Nil(t, result[0].OpensAt)

	require.Equal(t, closedSoonShop, result[1].ID)
	require.False(t, result[1].Opened)
	require.NotNil(t, result[1].OpensAt)
	require.Nil(t, result[1].ClosesAt)
}

func TestGetAllShops_NoScheduleKeepsOpenButHidesClosed(t *testing.T) {
	repo := mocksshops.NewShopsRepo(t)
	bcv := mocksbcv.NewBcvApp(t)
	shopsApp := app.NewShopsApp(repo, nil, bcv, nil)

	openShop := uuid.New()
	closedShopWithoutSchedule := uuid.New()

	query := dtos.ListShopsQuery{Latitude: 10.48, Longitude: -66.9}

	repo.On("GetAllShops", context.Background(), query).Return(dtos.ShopsResponse{
		{ID: openShop, Opened: true},
		{ID: closedShopWithoutSchedule, Opened: false},
	}, nil)
	bcv.On("GetRate", mock.Anything).Return(&bcvmodels.ExchangeRate{Rate: 40}, nil)
	repo.On("GetSchedulesByShopIDs", context.Background(), mock.Anything).
		Return(map[uuid.UUID][]dtos.ScheduleResponse{}, nil)

	result, err := shopsApp.GetAllShops(context.Background(), query)
	require.NoError(t, err)
	require.Len(t, result, 1)
	require.Equal(t, openShop, result[0].ID)
}

func TestGetShopsBy_AnnotatesScheduleWithoutHiding(t *testing.T) {
	repo := mocksshops.NewShopsRepo(t)
	bcv := mocksbcv.NewBcvApp(t)
	shopsApp := app.NewShopsApp(repo, nil, bcv, nil)

	now := bizclock.Now()
	closedFarShop := uuid.New()

	criteria := dtos.SearchShopsByID{ID: closedFarShop}
	coords := dtos.Coords{Latitude: 10.48, Longitude: -66.9}

	repo.On("GetShopsBy", context.Background(), criteria, coords).Return(dtos.ShopsResponse{
		{ID: closedFarShop, Opened: false},
	}, nil)
	bcv.On("GetRate", mock.Anything).Return(&bcvmodels.ExchangeRate{Rate: 40}, nil)
	repo.On("GetSchedulesByShopIDs", context.Background(), mock.Anything).Return(map[uuid.UUID][]dtos.ScheduleResponse{
		closedFarShop: {windowAround(now, 2*time.Hour, 3*time.Hour)},
	}, nil)

	result, err := shopsApp.GetShopsBy(context.Background(), criteria, coords)
	require.NoError(t, err)
	require.Len(t, result, 1)
	require.Equal(t, closedFarShop, result[0].ID)
	require.NotNil(t, result[0].OpensAt)
}
