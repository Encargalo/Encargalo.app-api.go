package repo_test

import (
	"testing"

	"Encargalo.app-api.go/internal/products/domain/models"
	"Encargalo.app-api.go/internal/products/repo"
	"Encargalo.app-api.go/internal/shops/domain/models/shops"
	"github.com/stretchr/testify/require"
)

func TestTrendingQuery_ProductionUsesShopsCoverageRadius(t *testing.T) {
	query, args := repo.TrendingQuery(true, 10.1780808, -66.8158868)

	require.Contains(t, query, ") <= ?")
	require.Equal(t, []any{models.AvailabilityHidden, 10.1780808, -66.8158868, 10.1780808, shops.CoverageRadiusKm}, args)
}

func TestTrendingQuery_OutsideProductionSkipsDistanceFilter(t *testing.T) {
	query, args := repo.TrendingQuery(false, 10.1780808, -66.8158868)

	require.NotContains(t, query, "radians")
	require.Equal(t, []any{models.AvailabilityHidden}, args)
}
