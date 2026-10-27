package repo

import (
	"context"
	"log/slog"
	"time"

	"Encargalo.app-api.go/internal/riders/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const tripOfferTimeoutsKey = "riders:trip_offer_timeouts"

// popDueScript saca atómicamente los miembros vencidos: el ZRANGEBYSCORE y el
// ZREM corren en un solo paso en Redis, así que dos réplicas que lo ejecuten
// al mismo tiempo nunca pueden reclamar el mismo trip_id (ver #359).
var popDueScript = redis.NewScript(`
local members = redis.call('ZRANGEBYSCORE', KEYS[1], '-inf', ARGV[1], 'LIMIT', 0, ARGV[2])
if #members > 0 then
	redis.call('ZREM', KEYS[1], unpack(members))
end
return members
`)

type tripOfferTimeoutRepo struct {
	redis *redis.Client
}

func NewTripOfferTimeoutRepo(redis *redis.Client) ports.TripOfferTimeoutStore {
	return &tripOfferTimeoutRepo{redis: redis}
}

func (r *tripOfferTimeoutRepo) Schedule(ctx context.Context, tripID uuid.UUID, at time.Time) error {
	err := r.redis.ZAdd(ctx, tripOfferTimeoutsKey, redis.Z{
		Score:  float64(at.UnixMilli()),
		Member: tripID.String(),
	}).Err()
	if err != nil {
		slog.Error("tripOfferTimeout Schedule redis failed", "tripID", tripID, "error", err)
		return errcustom.ErrUnexpectedError
	}
	return nil
}

func (r *tripOfferTimeoutRepo) Cancel(ctx context.Context, tripID uuid.UUID) error {
	if err := r.redis.ZRem(ctx, tripOfferTimeoutsKey, tripID.String()).Err(); err != nil {
		slog.Error("tripOfferTimeout Cancel redis failed", "tripID", tripID, "error", err)
		return errcustom.ErrUnexpectedError
	}
	return nil
}

func (r *tripOfferTimeoutRepo) PopDue(ctx context.Context, limit int64) ([]uuid.UUID, error) {
	now := float64(time.Now().UnixMilli())
	members, err := popDueScript.Run(ctx, r.redis, []string{tripOfferTimeoutsKey}, now, limit).StringSlice()
	if err != nil {
		slog.Error("tripOfferTimeout PopDue redis failed", "error", err)
		return nil, errcustom.ErrUnexpectedError
	}
	return parseTripIDs(members), nil
}

// ScheduleIfAbsent es un ZADD NX: si tripID ya tiene un vencimiento en cola no
// lo toca, para que el barrido de rescate no adelante una ronda que ya está
// corriendo su curso normal.
func (r *tripOfferTimeoutRepo) ScheduleIfAbsent(ctx context.Context, tripID uuid.UUID, at time.Time) error {
	err := r.redis.ZAddNX(ctx, tripOfferTimeoutsKey, redis.Z{
		Score:  float64(at.UnixMilli()),
		Member: tripID.String(),
	}).Err()
	if err != nil {
		slog.Error("tripOfferTimeout ScheduleIfAbsent redis failed", "tripID", tripID, "error", err)
		return errcustom.ErrUnexpectedError
	}
	return nil
}

func parseTripIDs(members []string) []uuid.UUID {
	out := make([]uuid.UUID, 0, len(members))
	for _, member := range members {
		id, err := uuid.Parse(member)
		if err != nil {
			slog.Error("tripOfferTimeout invalid member in queue", "member", member, "error", err)
			continue
		}
		out = append(out, id)
	}
	return out
}
