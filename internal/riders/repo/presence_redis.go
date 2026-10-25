package repo

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"strconv"
	"time"

	"Encargalo.app-api.go/internal/riders/domain/models"
	"Encargalo.app-api.go/internal/riders/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	presenceKeyPrefix = "riders:presence:"
	presenceGeoPrefix = "riders:geo:"
)

type riderPresenceRepo struct {
	redis *redis.Client
}

func NewRiderPresenceRepo(redis *redis.Client) ports.RiderPresenceStore {
	return &riderPresenceRepo{redis: redis}
}

func presenceKey(riderID uuid.UUID) string {
	return presenceKeyPrefix + riderID.String()
}

func presenceGeoKey(mode models.RiderMode) string {
	return presenceGeoPrefix + string(mode)
}

// otherMode es el índice del que hay que sacar al rider al guardar, para que un
// cambio de modo no lo deje listado en los dos.
func otherMode(mode models.RiderMode) models.RiderMode {
	if mode == models.RiderModeTrips {
		return models.RiderModeOrders
	}
	return models.RiderModeTrips
}

func (r *riderPresenceRepo) SaveLocation(ctx context.Context, presence models.RiderPresence) error {
	member := presence.RiderID.String()
	fields := map[string]any{
		"lat":        presence.Location.Latitude,
		"lng":        presence.Location.Longitude,
		"h3":         presence.Location.H3Cell,
		"updated_at": presence.Location.UpdatedAt.UnixMilli(),
		"mode":       string(presence.Mode),
		"vehicle":    presence.Vehicle,
	}

	pipe := r.redis.Pipeline()
	pipe.HSet(ctx, presenceKey(presence.RiderID), fields)
	pipe.Expire(ctx, presenceKey(presence.RiderID), models.PresenceTTL)
	pipe.GeoAdd(ctx, presenceGeoKey(presence.Mode), &redis.GeoLocation{
		Name:      member,
		Latitude:  presence.Location.Latitude,
		Longitude: presence.Location.Longitude,
	})
	pipe.ZRem(ctx, presenceGeoKey(otherMode(presence.Mode)), member)

	if _, err := pipe.Exec(ctx); err != nil {
		slog.Error("riderPresence SaveLocation redis failed", "riderID", presence.RiderID, "error", err)
		return errcustom.ErrUnexpectedError
	}
	return nil
}

func (r *riderPresenceRepo) Remove(ctx context.Context, riderID uuid.UUID) error {
	member := riderID.String()

	pipe := r.redis.Pipeline()
	pipe.Del(ctx, presenceKey(riderID))
	pipe.ZRem(ctx, presenceGeoKey(models.RiderModeTrips), member)
	pipe.ZRem(ctx, presenceGeoKey(models.RiderModeOrders), member)

	if _, err := pipe.Exec(ctx); err != nil {
		slog.Error("riderPresence Remove redis failed", "riderID", riderID, "error", err)
		return errcustom.ErrUnexpectedError
	}
	return nil
}

func (r *riderPresenceRepo) SearchNearby(ctx context.Context, query models.PresenceQuery) ([]models.RiderLocationEntry, error) {
	geoKey := presenceGeoKey(query.Mode)
	members, err := r.redis.GeoSearch(ctx, geoKey, &redis.GeoSearchQuery{
		Longitude:  query.Longitude,
		Latitude:   query.Latitude,
		Radius:     query.RadiusKm,
		RadiusUnit: "km",
		Sort:       "ASC",
	}).Result()
	if err != nil {
		slog.Error("riderPresence SearchNearby geo failed", "error", err)
		return nil, errcustom.ErrUnexpectedError
	}
	if len(members) == 0 {
		return []models.RiderLocationEntry{}, nil
	}

	hashes, err := r.loadPresences(ctx, members)
	if err != nil {
		return nil, err
	}

	cutoff := time.Now().Add(-query.MaxAge)
	out := make([]models.RiderLocationEntry, 0, len(members))
	stale := make([]any, 0, len(members))

	for i, member := range members {
		// El hash expiró pero el miembro sigue en el índice: GEO no tiene TTL por miembro.
		if len(hashes[i]) == 0 {
			stale = append(stale, member)
			continue
		}
		presence, ok := decodePresence(member, hashes[i])
		if !ok {
			continue
		}
		// El índice quedó desfasado del modo real: sacarlo lo repara solo.
		if presence.Mode != query.Mode {
			stale = append(stale, member)
			continue
		}
		if presence.Location.UpdatedAt.Before(cutoff) {
			continue
		}
		if slices.Contains(query.ExcludeVehicles, presence.Vehicle) {
			continue
		}
		if slices.Contains(query.ExcludeRiders, presence.RiderID) {
			continue
		}
		out = append(out, models.RiderLocationEntry{RiderID: presence.RiderID, Location: presence.Location})
	}

	r.dropStale(ctx, geoKey, stale)

	if query.Limit > 0 && len(out) > query.Limit {
		out = out[:query.Limit]
	}
	return out, nil
}

// loadPresences trae los hashes de los miembros devueltos por GEOSEARCH en un
// solo viaje, no uno por rider.
func (r *riderPresenceRepo) loadPresences(ctx context.Context, members []string) ([]map[string]string, error) {
	pipe := r.redis.Pipeline()
	cmds := make([]*redis.MapStringStringCmd, len(members))
	for i, member := range members {
		cmds[i] = pipe.HGetAll(ctx, presenceKeyPrefix+member)
	}

	// HGETALL devuelve hash vacío, no redis.Nil; cualquier otro error sí aborta.
	if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) {
		slog.Error("riderPresence loadPresences redis failed", "error", err)
		return nil, errcustom.ErrUnexpectedError
	}

	out := make([]map[string]string, len(members))
	for i, cmd := range cmds {
		out[i] = cmd.Val()
	}
	return out, nil
}

// dropStale saca del índice a los miembros que ya no tienen presencia, para no
// volver a leerlos en cada búsqueda.
func (r *riderPresenceRepo) dropStale(ctx context.Context, geoKey string, members []any) {
	if len(members) == 0 {
		return
	}
	if err := r.redis.ZRem(ctx, geoKey, members...).Err(); err != nil {
		slog.Error("riderPresence dropStale redis failed", "error", err)
	}
}

func decodePresence(member string, fields map[string]string) (models.RiderPresence, bool) {
	riderID, err := uuid.Parse(member)
	if err != nil {
		return models.RiderPresence{}, false
	}
	lat, errLat := strconv.ParseFloat(fields["lat"], 64)
	lng, errLng := strconv.ParseFloat(fields["lng"], 64)
	if errLat != nil || errLng != nil {
		return models.RiderPresence{}, false
	}
	updated, err := strconv.ParseInt(fields["updated_at"], 10, 64)
	if err != nil {
		return models.RiderPresence{}, false
	}
	return models.RiderPresence{
		RiderID: riderID,
		Location: models.RiderLocation{
			Latitude:  lat,
			Longitude: lng,
			H3Cell:    fields["h3"],
			UpdatedAt: time.UnixMilli(updated),
		},
		Mode:    models.RiderMode(fields["mode"]),
		Vehicle: fields["vehicle"],
	}, true
}
