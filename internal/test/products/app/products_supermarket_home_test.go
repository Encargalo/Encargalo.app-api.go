package app_test

import (
	"context"
	"testing"
	"time"

	bcvmodels "Encargalo.app-api.go/internal/bcv/domain/models"
	"Encargalo.app-api.go/internal/products/app"
	"Encargalo.app-api.go/internal/products/domain/dtos"
	"Encargalo.app-api.go/internal/products/domain/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func candidate(macroName, categoryName string) models.SupermarketCategoryCandidate {
	return models.SupermarketCategoryCandidate{
		MacroID:      uuid.New().String(),
		MacroName:    macroName,
		MacroSlug:    macroName,
		CategoryName: categoryName,
	}
}

// pool10 son 10 candidatas ordenables de forma estable (por MacroName, luego
// CategoryName): "Bebidas/Agua" .. "Bebidas/Jugos". SelectDailyShelves las
// ordena igual internamente, así que el índice i aquí coincide con el índice
// post-sort.
func pool10() []models.SupermarketCategoryCandidate {
	names := []string{"Agua", "Bebidas Light", "Gaseosas", "Jugos", "Licores y Cervezas", "Malta", "Refrescos", "Te", "Vino", "Whisky"}
	pool := make([]models.SupermarketCategoryCandidate, len(names))
	for i, n := range names {
		pool[i] = candidate("Bebidas", n)
	}
	return pool
}

func TestSelectDailyShelves_EmptyPoolReturnsNil(t *testing.T) {
	got := app.SelectDailyShelves(nil, time.Now(), 8)
	assert.Nil(t, got)
}

func TestSelectDailyShelves_RotatesWindowByDayOfYear(t *testing.T) {
	pool := pool10()
	// 2024-01-01 es día 1 del año -> start = 1 % 10 = 1 -> ventana [1,2,3].
	day1 := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	got := app.SelectDailyShelves(pool, day1, 3)
	require.Len(t, got, 3)
	assert.Equal(t, []string{"Bebidas Light", "Gaseosas", "Jugos"}, []string{got[0].CategoryName, got[1].CategoryName, got[2].CategoryName})

	// 2024-01-02 es día 2 -> start = 2 % 10 = 2 -> ventana [2,3,4], distinta.
	day2 := time.Date(2024, 1, 2, 12, 0, 0, 0, time.UTC)
	got2 := app.SelectDailyShelves(pool, day2, 3)
	require.Len(t, got2, 3)
	assert.Equal(t, []string{"Gaseosas", "Jugos", "Licores y Cervezas"}, []string{got2[0].CategoryName, got2[1].CategoryName, got2[2].CategoryName})
}

func TestSelectDailyShelves_WrapsAroundPool(t *testing.T) {
	pool := pool10()
	// día 9 -> start=9, ventana de 3 con wraparound: índices 9,0,1.
	day9 := time.Date(2024, 1, 9, 12, 0, 0, 0, time.UTC)
	got := app.SelectDailyShelves(pool, day9, 3)
	require.Len(t, got, 3)
	assert.Equal(t, []string{"Whisky", "Agua", "Bebidas Light"}, []string{got[0].CategoryName, got[1].CategoryName, got[2].CategoryName})
}

func TestSelectDailyShelves_CountGreaterThanPoolReturnsWholePool(t *testing.T) {
	pool := pool10()
	got := app.SelectDailyShelves(pool, time.Now(), 50)
	assert.Len(t, got, len(pool))
}

func TestSelectDailyShelves_PinsLicoresOnWeekend(t *testing.T) {
	pool := pool10()
	// 2024-01-01 fue lunes; busco el viernes siguiente (2024-01-05, día 5).
	// start = 5 % 10 = 5 -> ventana [5,6,7] = Malta, Refrescos, Te — no
	// incluye "Licores y Cervezas" (índice 4), así que debe agregarse fijo.
	friday := time.Date(2024, 1, 5, 12, 0, 0, 0, time.UTC)
	require.Equal(t, time.Friday, friday.Weekday())

	got := app.SelectDailyShelves(pool, friday, 3)

	require.Len(t, got, 4)
	names := make([]string, len(got))
	for i, c := range got {
		names[i] = c.CategoryName
	}
	assert.Contains(t, names, "Licores y Cervezas")
}

func TestSelectDailyShelves_NoPinOnWeekday(t *testing.T) {
	pool := pool10()
	// 2024-01-01 es lunes; misma ventana [1,2,3] = Cervezas, Gaseosas, Jugos.
	monday := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	require.Equal(t, time.Monday, monday.Weekday())

	got := app.SelectDailyShelves(pool, monday, 3)

	require.Len(t, got, 3)
	for _, c := range got {
		assert.NotEqual(t, "Licores y Cervezas", c.CategoryName)
	}
}

func TestSelectDailyShelves_WeekendDoesNotDuplicateLicoresAlreadyRotatedIn(t *testing.T) {
	pool := pool10()
	// día 12 (viernes) -> start=12%10=2 -> ventana [2,3,4] = Gaseosas, Jugos,
	// "Licores y Cervezas": la rotación ya la incluyó, así que el pin de fin
	// de semana no debe duplicarla.
	day12 := time.Date(2024, 1, 12, 12, 0, 0, 0, time.UTC)
	require.Equal(t, time.Friday, day12.Weekday())

	got := app.SelectDailyShelves(pool, day12, 3)

	require.Len(t, got, 3)
	count := 0
	for _, c := range got {
		if c.CategoryName == "Licores y Cervezas" {
			count++
		}
	}
	assert.Equal(t, 1, count)
}

func TestGetSupermarketHome_PoolEmpty_ReturnsEmptyWithoutQueryingEntries(t *testing.T) {
	svc, m := newProductAppFixture(t)
	coords := dtos.SupermarketHomeCoords{Latitude: 10.1, Longitude: -66.8}

	m.repo.On("SearchSupermarketCategoryPool", mock.Anything, coords).Return(nil, nil)

	got, err := svc.GetSupermarketHome(context.Background(), coords)

	require.NoError(t, err)
	assert.Empty(t, got)
	m.repo.AssertNotCalled(t, "SearchSupermarketHomeEntries", mock.Anything, mock.Anything, mock.Anything)
}

func TestGetSupermarketHome_PoolError_Propagates(t *testing.T) {
	svc, m := newProductAppFixture(t)
	coords := dtos.SupermarketHomeCoords{Latitude: 10.1, Longitude: -66.8}

	m.repo.On("SearchSupermarketCategoryPool", mock.Anything, coords).Return(nil, assert.AnError)

	_, err := svc.GetSupermarketHome(context.Background(), coords)

	assert.ErrorIs(t, err, assert.AnError)
}

func TestGetSupermarketHome_EntriesError_Propagates(t *testing.T) {
	svc, m := newProductAppFixture(t)
	coords := dtos.SupermarketHomeCoords{Latitude: 10.1, Longitude: -66.8}
	pool := pool10()

	m.repo.On("SearchSupermarketCategoryPool", mock.Anything, coords).Return(pool, nil)
	m.repo.On("SearchSupermarketHomeEntries", mock.Anything, coords, mock.Anything).Return(nil, assert.AnError)

	_, err := svc.GetSupermarketHome(context.Background(), coords)

	assert.ErrorIs(t, err, assert.AnError)
}

func TestGetSupermarketHome_GroupsCapsFiltersScheduleAndAppliesRate(t *testing.T) {
	svc, m := newProductAppFixture(t)
	coords := dtos.SupermarketHomeCoords{Latitude: 10.1, Longitude: -66.8}

	// Pool del mismo tamaño que el tope de repisas: cualquier rotación (sea
	// cual sea "hoy" al correr el test) selecciona el pool completo, así el
	// test no depende de la fecha real.
	macroA := candidate("Viveres", "Arroz y Granos")
	macroB := candidate("Limpieza", "Detergentes")
	pool := []models.SupermarketCategoryCandidate{macroA, macroB}
	for range 6 {
		pool = append(pool, candidate("Otros", "Relleno"))
	}

	visibleEntry := dtos.HomeEntry{
		CategoryName: "Arroz y Granos",
		Shop:         dtos.HomeShop{ID: uuid.New(), Name: "Super A", DeliveryFee: 2},
		Product:      dtos.HomeProduct{ID: uuid.New(), Name: "Arroz 1kg", Ref: 2},
	}
	hiddenByScheduleEntry := dtos.HomeEntry{
		CategoryName: "Detergentes",
		Shop:         dtos.HomeShop{ID: uuid.New(), Name: "Super B", DeliveryFee: 1.5},
		Product:      dtos.HomeProduct{ID: uuid.New(), Name: "Jabon liquido", Ref: 3},
		Schedules:    []dtos.ScheduleResponse{{StartTime: "00:00:00", EndTime: "23:59:59", Enabled: false}},
	}
	entries := dtos.HomeEntriesResponse{visibleEntry, hiddenByScheduleEntry}

	m.repo.On("SearchSupermarketCategoryPool", mock.Anything, coords).Return(pool, nil)
	m.repo.On("SearchSupermarketHomeEntries", mock.Anything, coords, mock.Anything).Return(entries, nil)
	m.bcv.On("GetRate", mock.Anything).Return(&bcvmodels.ExchangeRate{Rate: 40}, nil)

	got, err := svc.GetSupermarketHome(context.Background(), coords)

	require.NoError(t, err)
	require.Len(t, got, 1, "Limpieza/Detergentes quedó sin productos tras el filtro de horario, así que no debe aparecer")
	assert.Equal(t, "Viveres", got[0].MacroCategory.Name)
	require.Len(t, got[0].MicroCategories, 1)
	assert.Equal(t, "Arroz y Granos", got[0].MicroCategories[0].Name)
	require.Len(t, got[0].MicroCategories[0].Products, 1)
	product := got[0].MicroCategories[0].Products[0]
	assert.Equal(t, "Arroz 1kg", product.Name)
	assert.Equal(t, 80.0, product.Bs) // 2 * 40
	assert.Equal(t, "Super A", product.Shop.Name)
	assert.Equal(t, 80.0, product.Shop.DeliveryFeeBs) // 2 * 40
}

func TestGetSupermarketHome_CapsProductsPerShelfToFinalLimit(t *testing.T) {
	svc, m := newProductAppFixture(t)
	coords := dtos.SupermarketHomeCoords{Latitude: 10.1, Longitude: -66.8}

	macroA := candidate("Viveres", "Arroz y Granos")
	pool := []models.SupermarketCategoryCandidate{macroA}
	for range 7 {
		pool = append(pool, candidate("Otros", "Relleno"))
	}

	entries := make(dtos.HomeEntriesResponse, 0, 20)
	for range 20 {
		entries = append(entries, dtos.HomeEntry{
			CategoryName: "Arroz y Granos",
			Shop:         dtos.HomeShop{ID: uuid.New()},
			Product:      dtos.HomeProduct{ID: uuid.New(), Ref: 1},
		})
	}

	m.repo.On("SearchSupermarketCategoryPool", mock.Anything, coords).Return(pool, nil)
	m.repo.On("SearchSupermarketHomeEntries", mock.Anything, coords, mock.Anything).Return(entries, nil)
	m.bcv.On("GetRate", mock.Anything).Return(&bcvmodels.ExchangeRate{Rate: 1}, nil)

	got, err := svc.GetSupermarketHome(context.Background(), coords)

	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Len(t, got[0].MicroCategories, 1)
	assert.Len(t, got[0].MicroCategories[0].Products, 15, "debe recortar al tope final por repisa")
}
