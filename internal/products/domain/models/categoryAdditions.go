package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type CategoryAddition struct {
	bun.BaseModel `bun:"table:products.categories_adiciones"`

	ID         uuid.UUID `bun:"id,pk,type:uuid,default:uuid_generate_v4()"`
	ShopID     uuid.UUID `bun:"shop_id,notnull"`
	CategoryID uuid.UUID `bun:"category_id,notnull"`
	AdditionID uuid.UUID `bun:"addition_id,notnull"`
	CreatedAt  time.Time `bun:"created_at,default:now()"`
	UpdatedAt  time.Time `bun:"updated_at,default:now()"`
	DeletedAt  time.Time `bun:"deleted_at,nullzero"`

	Addition *Addition `bun:"rel:belongs-to,join:addition_id=id"`
}
