package repo

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"Encargalo.app-api.go/internal/analytics/domain/models"
	"Encargalo.app-api.go/internal/analytics/domain/ports"
	"Encargalo.app-api.go/internal/shared/adapters/postgres"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/uptrace/bun"
)

type syncLogRepo struct {
	db *bun.DB
}

func NewSyncLogRepo(olap *postgres.OLAPDB) ports.SyncLogRepo {
	return &syncLogRepo{db: olap.DB}
}

func (r *syncLogRepo) GetLastSyncedTo(ctx context.Context, entity models.ETLEntity) (*time.Time, error) {
	var log models.SyncLog
	err := r.db.NewSelect().Model(&log).Where("entity = ?", entity).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, errcustom.Unexpected(err)
	}
	return &log.LastSyncedTo, nil
}

func (r *syncLogRepo) Upsert(ctx context.Context, entity models.ETLEntity, syncedTo time.Time) error {
	now := time.Now().UTC()
	row := models.SyncLog{
		Entity:       entity,
		LastSyncedTo: syncedTo,
		UpdatedAt:    now,
	}
	_, err := r.db.NewInsert().
		Model(&row).
		On("CONFLICT (entity) DO UPDATE").
		Set("last_synced_to = EXCLUDED.last_synced_to").
		Set("updated_at = EXCLUDED.updated_at").
		Exec(ctx)
	if err != nil {
		return errcustom.Unexpected(err)
	}
	return nil
}
