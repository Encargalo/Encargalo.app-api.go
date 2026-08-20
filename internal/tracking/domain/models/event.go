package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Event struct {
	bun.BaseModel `bun:"table:tracking.events"`

	ID        uuid.UUID      `bun:"id,pk,type:uuid"`
	Type      string         `bun:"type,notnull"`
	Params    map[string]any `bun:"params,type:jsonb,notnull"`
	CreatedAt time.Time      `bun:"created_at,notnull"`
}
