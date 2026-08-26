package app_test

import (
	"context"
	"testing"

	"Encargalo.app-api.go/internal/bcv/app"
	"Encargalo.app-api.go/internal/bcv/domain/models"
	mocksbcv "Encargalo.app-api.go/internal/mocks/bcv/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/stretchr/testify/require"
)

// Solo GetRate es testeable de forma determinista sin red: New/Start disparan
// goroutines, un cron real y scraping HTTP contra BCV; fetchOnStartup,
// syncHolidays, runDailyCron y trySyncUpdatedRate son métodos no exportados
// solo alcanzables desde ahí (mismo criterio de los workers con goroutine que
// se dejaron sin cubrir en orders/users/shops).
func TestGetRate_DelegatesToCache(t *testing.T) {
	cache := mocksbcv.NewBcvCache(t)
	svc := app.New(nil, nil, cache, nil)

	cache.On("GetLatest", context.Background()).Return(&models.ExchangeRate{Rate: 45.5}, nil)

	rate, err := svc.GetRate(context.Background())

	require.NoError(t, err)
	require.Equal(t, 45.5, rate.Rate)
}

func TestGetRate_CacheError(t *testing.T) {
	cache := mocksbcv.NewBcvCache(t)
	svc := app.New(nil, nil, cache, nil)

	cache.On("GetLatest", context.Background()).Return(nil, errcustom.ErrNotFound)

	_, err := svc.GetRate(context.Background())

	require.ErrorIs(t, err, errcustom.ErrNotFound)
}
