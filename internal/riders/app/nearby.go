package app

import (
	"context"
	"math"

	"Encargalo.app-api.go/internal/pkg/geo"
	riderModels "Encargalo.app-api.go/internal/riders/domain/models"
	ridersPorts "Encargalo.app-api.go/internal/riders/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	tripsDtos "Encargalo.app-api.go/internal/trips/domain/dtos"
	"github.com/google/uuid"
)

func (r *appRidersManager) NearbyRiders(ctx context.Context, lat, lng float64) ([]tripsDtos.NearbyRider, error) {
	// InVenezuela acota el área de servicio; Redis acepta cualquier coordenada del planeta.
	if !geo.InVenezuela(lat, lng) {
		return nil, errcustom.ErrInvalidCoordinates
	}
	if r.presence == nil {
		return []tripsDtos.NearbyRider{}, nil
	}

	entries, err := r.presence.SearchNearby(ctx, riderModels.PresenceQuery{
		Latitude:  lat,
		Longitude: lng,
		RadiusKm:  geo.RingsToKm(riderModels.NearbyRiderRings),
		Mode:      riderModels.RiderModeTrips,
		MaxAge:    riderModels.NearbyRiderMaxAge,
		// La bici no lleva pasajeros, igual que en el matching; sin vehículo cargado no excluye.
		ExcludeVehicles: []string{riderModels.VehicleTypeBike},
	})
	if err != nil {
		return nil, err
	}

	return r.pinsForAvailable(entries), nil
}

// AvgPickupDistanceKm usa el mismo criterio de candidatos que el matching real
// (connectedCandidates: conectados, tope de tripOfferCandidateCount) para que
// el cargo de acercamiento en la cotización refleje a quién se le ofertaría
// de verdad el viaje, no a cualquier rider con presencia reciente.
func (r *appRidersManager) AvgPickupDistanceKm(ctx context.Context, lat, lng float64) (float64, bool) {
	if r.presence == nil {
		return 0, false
	}

	entries, err := r.presence.SearchNearby(ctx, riderModels.PresenceQuery{
		Latitude:        lat,
		Longitude:       lng,
		RadiusKm:        geo.RingsToKm(tripSearchMaxRings),
		Mode:            riderModels.RiderModeTrips,
		MaxAge:          riderModels.TripMatchRiderMaxAge,
		ExcludeVehicles: []string{riderModels.VehicleTypeBike},
	})
	if err != nil {
		return 0, false
	}

	candidates := r.connectedCandidates(entries)
	if len(candidates) == 0 {
		return 0, false
	}

	sum := 0.0
	for _, c := range candidates {
		sum += geo.HaversineKm(c.Location.Latitude, c.Location.Longitude, lat, lng)
	}
	return sum / float64(len(candidates)), true
}

// pinsForAvailable descarta a los riders con viaje asignado y corta en el tope
// del mapa. El tope no viaja en la query porque se aplica después de ese filtro.
func (r *appRidersManager) pinsForAvailable(entries []riderModels.RiderLocationEntry) []tripsDtos.NearbyRider {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]tripsDtos.NearbyRider, 0, riderModels.NearbyRiderMaxResults)
	for _, entry := range entries {
		// riderTrips sigue en memoria hasta que el estado de asignación salga del proceso.
		if _, busy := r.riderTrips[entry.RiderID]; busy {
			continue
		}
		out = append(out, tripsDtos.NearbyRider{
			Latitude:  roundMapCoord(entry.Location.Latitude),
			Longitude: roundMapCoord(entry.Location.Longitude),
		})
		if len(out) == riderModels.NearbyRiderMaxResults {
			return out
		}
	}
	return out
}

func roundMapCoord(v float64) float64 {
	return math.Round(v*riderModels.NearbyCoordDecimals) / riderModels.NearbyCoordDecimals
}

// MarkRiderBusyForTest marca al rider como si tuviera un viaje asignado.
func MarkRiderBusyForTest(svc ridersPorts.RidersApp, riderID uuid.UUID) {
	mgr := svc.(*appRidersManager)
	mgr.mu.Lock()
	defer mgr.mu.Unlock()
	mgr.riderTrips[riderID] = riderAssignment{tripID: uuid.New(), customerID: uuid.New()}
}
