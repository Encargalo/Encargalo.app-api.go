package repo

import (
	"context"

	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/tracking/domain/models"
	"Encargalo.app-api.go/internal/tracking/domain/ports"
	"github.com/uptrace/bun"
)

type trackingRepo struct {
	db *bun.DB
}

func NewTrackingRepo(db *bun.DB) ports.TrackingRepo {
	return &trackingRepo{db: db}
}

func (r *trackingRepo) Insert(ctx context.Context, event models.Event) error {
	if event.Params == nil {
		event.Params = map[string]any{}
	}
	_, err := r.db.NewInsert().
		Model(&event).
		On("CONFLICT (id) DO NOTHING").
		Exec(ctx)
	if err != nil {
		return errcustom.Unexpected(err)
	}
	return nil
}
