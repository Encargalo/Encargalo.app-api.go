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

const shipmentOfferTimeoutsKey = "riders:shipment_offer_timeouts"

type shipmentOfferTimeoutRepo struct {
	redis *redis.Client
}

func NewShipmentOfferTimeoutRepo(redis *redis.Client) ports.ShipmentOfferTimeoutStore {
	return &shipmentOfferTimeoutRepo{redis: redis}
}

func (r *shipmentOfferTimeoutRepo) Schedule(ctx context.Context, shipmentID uuid.UUID, at time.Time) error {
	err := r.redis.ZAdd(ctx, shipmentOfferTimeoutsKey, redis.Z{
		Score:  float64(at.UnixMilli()),
		Member: shipmentID.String(),
	}).Err()
	if err != nil {
		slog.Error("shipmentOfferTimeout Schedule redis failed", "shipmentID", shipmentID, "error", err)
		return errcustom.ErrUnexpectedError
	}
	return nil
}

func (r *shipmentOfferTimeoutRepo) Cancel(ctx context.Context, shipmentID uuid.UUID) error {
	if err := r.redis.ZRem(ctx, shipmentOfferTimeoutsKey, shipmentID.String()).Err(); err != nil {
		slog.Error("shipmentOfferTimeout Cancel redis failed", "shipmentID", shipmentID, "error", err)
		return errcustom.ErrUnexpectedError
	}
	return nil
}

func (r *shipmentOfferTimeoutRepo) PopDue(ctx context.Context, limit int64) ([]uuid.UUID, error) {
	now := float64(time.Now().UnixMilli())
	members, err := popDueScript.Run(ctx, r.redis, []string{shipmentOfferTimeoutsKey}, now, limit).StringSlice()
	if err != nil {
		slog.Error("shipmentOfferTimeout PopDue redis failed", "error", err)
		return nil, errcustom.ErrUnexpectedError
	}
	return parseTripIDs(members), nil
}

// ScheduleIfAbsent es un ZADD NX: si shipmentID ya tiene un vencimiento en
// cola no lo toca, para que el barrido de rescate no adelante uno que ya
// está esperando el suyo.
func (r *shipmentOfferTimeoutRepo) ScheduleIfAbsent(ctx context.Context, shipmentID uuid.UUID, at time.Time) error {
	err := r.redis.ZAddNX(ctx, shipmentOfferTimeoutsKey, redis.Z{
		Score:  float64(at.UnixMilli()),
		Member: shipmentID.String(),
	}).Err()
	if err != nil {
		slog.Error("shipmentOfferTimeout ScheduleIfAbsent redis failed", "shipmentID", shipmentID, "error", err)
		return errcustom.ErrUnexpectedError
	}
	return nil
}
