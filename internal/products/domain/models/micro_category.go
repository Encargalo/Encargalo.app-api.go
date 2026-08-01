package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type MicroCategories []MicroCategory

type MicroCategory struct {
	bun.BaseModel `bun:"table:products.micro_categories" swaggerignore:"true"`

	ID              uuid.UUID `bun:"id,pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	MacroCategoryID uuid.UUID `bun:"macro_category_id,notnull" json:"macro_category_id"`
	Name            string    `bun:"name" json:"name"`
	Slug            string    `bun:"slug" json:"slug"`
	CreatedAt       time.Time `bun:"created_at,default:now()" json:"-"`
}
