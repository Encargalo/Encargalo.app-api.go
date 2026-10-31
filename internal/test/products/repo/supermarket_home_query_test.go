package repo_test

import (
	"testing"

	"Encargalo.app-api.go/internal/products/domain/models"
	"Encargalo.app-api.go/internal/products/repo"
	"Encargalo.app-api.go/internal/shops/domain/models/shops"
	"github.com/stretchr/testify/require"
)

func TestSupermarketCategoryPoolQuery_ProductionUsesShopsCoverageRadius(t *testing.T) {
	query, args := repo.SupermarketCategoryPoolQuery(true, 10.1780808, -66.8158868)

	require.Contains(t, query, ") <= ?")
	require.Contains(t, query, "s.type = ?")
	require.Equal(t, []any{shops.ShopTypeSupermarket, models.AvailabilityHidden, models.AvailabilityHidden, 10.1780808, -66.8158868, 10.1780808, shops.CoverageRadiusKm}, args)
}

func TestSupermarketCategoryPoolQuery_OutsideProductionSkipsDistanceFilter(t *testing.T) {
	query, args := repo.SupermarketCategoryPoolQuery(false, 10.1780808, -66.8158868)

	require.NotContains(t, query, "radians")
	require.Equal(t, []any{shops.ShopTypeSupermarket, models.AvailabilityHidden, models.AvailabilityHidden}, args)
}

func TestSupermarketHomeProductsQuery_BuildsInClauseAndArgsInOrder(t *testing.T) {
	names := []string{"Arroz y Granos", "Detergentes"}
	query, args := repo.SupermarketHomeProductsQuery(false, 10.1, -66.8, names, 4)

	require.Contains(t, query, "c.name IN (?,?)")
	require.Contains(t, query, "shop_rank <= ?")
	require.Equal(t, []any{
		shops.ShopTypeSupermarket, models.AvailabilityHidden, models.AvailabilityHidden,
		"Arroz y Granos", "Detergentes",
		4,
	}, args)
}

func TestSupermarketHomeProductsQuery_ProductionAddsDistanceFilterBeforeShopRank(t *testing.T) {
	names := []string{"Arroz y Granos"}
	query, args := repo.SupermarketHomeProductsQuery(true, 10.1, -66.8, names, 4)

	require.Contains(t, query, "radians")
	require.Equal(t, []any{
		shops.ShopTypeSupermarket, models.AvailabilityHidden, models.AvailabilityHidden,
		"Arroz y Granos",
		10.1, -66.8, 10.1, shops.CoverageRadiusKm,
		4,
	}, args)
}
