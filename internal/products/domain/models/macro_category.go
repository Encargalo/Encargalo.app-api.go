package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type MacroCategory struct {
	bun.BaseModel `bun:"table:products.macro_categories" swaggerignore:"true"`

	ID        uuid.UUID `bun:"id,pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	Name      string    `bun:"name" json:"name"`
	Slug      string    `bun:"slug" json:"slug"`
	CreatedAt time.Time `bun:"created_at,default:now()" json:"-"`
}
