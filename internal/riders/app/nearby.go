package app

import (
	"math"
	"sort"
	"time"

	"Encargalo.app-api.go/internal/pkg/geo"
	riderModels "Encargalo.app-api.go/internal/riders/domain/models"
	ridersPorts "Encargalo.app-api.go/internal/riders/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	tripsDtos "Encargalo.app-api.go/internal/trips/domain/dtos"
	"github.com/google/uuid"
)

func (r *appRidersManager) NearbyRiders(lat, lng float64) ([]tripsDtos.NearbyRider, error) {
	origin, err := geo.LatLngToCell(lat, lng, geo.DriverMatchResolution)
	if err != nil {
		return nil, errcustom.ErrInvalidCoordinates
	}
	cells, err := geo.GridDisk(origin, riderModels.NearbyRiderRings)
	if err != nil {
		return nil, errcustom.ErrUnexpectedError
	}

	now := time.Now()
	type candidate struct {
		lat, lng, km float64
	}

	r.mu.RLock()
	matched := make([]candidate, 0, riderModels.NearbyRiderMaxResults)
	for _, cell := range cells {
		for riderID := range r.ridersByCell[cell] {
			loc, ok := r.freshTripRiderLocked(riderID, now)
			if !ok {
				continue
			}
			matched = append(matched, candidate{
				lat: loc.Latitude,
				lng: loc.Longitude,
				km:  geo.HaversineKm(lat, lng, loc.Latitude, loc.Longitude),
			})
		}
	}
	r.mu.RUnlock()

	sort.Slice(matched, func(i, j int) bool { return matched[i].km < matched[j].km })
	if len(matched) > riderModels.NearbyRiderMaxResults {
		matched = matched[:riderModels.NearbyRiderMaxResults]
	}

	out := make([]tripsDtos.NearbyRider, len(matched))
	for i, c := range matched {
		out[i] = tripsDtos.NearbyRider{
			Latitude:  roundMapCoord(c.lat),
			Longitude: roundMapCoord(c.lng),
		}
	}
	return out, nil
}

func (r *appRidersManager) freshTripRiderLocked(riderID uuid.UUID, now time.Time) (riderModels.RiderLocation, bool) {
	if r.riderMode[riderID] != riderModels.RiderModeTrips {
		return riderModels.RiderLocation{}, false
	}
	if _, busy := r.riderTrips[riderID]; busy {
		return riderModels.RiderLocation{}, false
	}
	loc, ok := r.riderLocation[riderID]
	if !ok || now.Sub(loc.UpdatedAt) > riderModels.NearbyRiderMaxAge {
		return riderModels.RiderLocation{}, false
	}
	return loc, true
}

func (r *appRidersManager) reindexRiderLocked(riderID uuid.UUID, prevCell, cell string, inTrips bool) {
	if prevCell != "" {
		r.unindexCellLocked(riderID, prevCell)
	}
	if !inTrips {
		if cell != "" && cell != prevCell {
			r.unindexCellLocked(riderID, cell)
		}
		return
	}
	if cell == "" {
		return
	}
	bucket := r.ridersByCell[cell]
	if bucket == nil {
		bucket = make(map[uuid.UUID]struct{})
		r.ridersByCell[cell] = bucket
	}
	bucket[riderID] = struct{}{}
}

func (r *appRidersManager) unindexCellLocked(riderID uuid.UUID, cell string) {
	if cell == "" || r.ridersByCell == nil {
		return
	}
	bucket := r.ridersByCell[cell]
	if bucket == nil {
		return
	}
	delete(bucket, riderID)
	if len(bucket) == 0 {
		delete(r.ridersByCell, cell)
	}
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

// AgeRiderLocationForTest envejece la última ubicación del rider.
func AgeRiderLocationForTest(svc ridersPorts.RidersApp, riderID uuid.UUID, age time.Duration) {
	mgr := svc.(*appRidersManager)
	mgr.mu.Lock()
	defer mgr.mu.Unlock()
	loc, ok := mgr.riderLocation[riderID]
	if !ok {
		return
	}
	loc.UpdatedAt = time.Now().Add(-age)
	mgr.riderLocation[riderID] = loc
}
