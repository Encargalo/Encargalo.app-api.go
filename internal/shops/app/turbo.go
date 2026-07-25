package app

import (
	"context"

	"Encargalo.app-api.go/internal/pkg/geo"
	"Encargalo.app-api.go/internal/shops/domain/dtos"
	shopsmodels "Encargalo.app-api.go/internal/shops/domain/models/shops"
)

func (s *shopsApp) GetTurboEntry(ctx context.Context, coords dtos.Coords) (dtos.TurboEntryResponse, error) {
	candidates, err := s.repo.ListTurboCandidates(ctx)
	if err != nil {
		return dtos.TurboEntryResponse{}, err
	}

	nearest, _, found := SelectNearestOpenTurboNode(candidates, coords.Latitude, coords.Longitude)
	if !found {
		return dtos.TurboEntryResponse{Available: false}, nil
	}

	shop, err := s.repo.ResolveTurboShop(ctx, nearest.ID, coords)
	if err != nil {
		return dtos.TurboEntryResponse{}, err
	}

	eta := shop.Time
	return dtos.TurboEntryResponse{
		Available:  true,
		Shop:       &shop,
		EtaMinutes: &eta,
		CoverageKm: shopsmodels.TurboCoverageRadiusKm,
	}, nil
}

// SelectNearestOpenTurboNode elige, entre los candidatos Turbo, el abierto
// más cercano dentro de TurboCoverageRadiusKm (empate: mayor score, luego
// menor ID). Exportada como función pura para poder testearla sin depender
// de la base de datos (ver ValidateTurboCoverage en orders/app por el mismo patrón).
func SelectNearestOpenTurboNode(candidates dtos.ShopsResponse, lat, lon float64) (dtos.ShopResponse, float64, bool) {
	var best dtos.ShopResponse
	var bestDistance float64
	found := false

	for _, candidate := range candidates {
		if !candidate.Opened {
			continue
		}
		distance := geo.HaversineKm(lat, lon, candidate.Latitude, candidate.Longitude)
		if distance > shopsmodels.TurboCoverageRadiusKm {
			continue
		}
		if isBetterTurboCandidate(candidate, distance, best, bestDistance, found) {
			best = candidate
			bestDistance = distance
			found = true
		}
	}

	return best, bestDistance, found
}

func isBetterTurboCandidate(candidate dtos.ShopResponse, distance float64, best dtos.ShopResponse, bestDistance float64, found bool) bool {
	if !found {
		return true
	}
	if distance < bestDistance {
		return true
	}
	if distance > bestDistance {
		return false
	}
	if candidate.Score > best.Score {
		return true
	}
	if candidate.Score < best.Score {
		return false
	}
	return candidate.ID.String() < best.ID.String()
}
