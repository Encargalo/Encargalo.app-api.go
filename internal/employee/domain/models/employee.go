package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Account struct {
	bun.BaseModel `bun:"table:employees.accounts"`

	ID        uuid.UUID  `bun:"id,pk,type:uuid,default:uuid_generate_v4()"`
	ShopID    uuid.UUID  `bun:"shop_id,type:uuid,notnull"`
	Name      string     `bun:"name,notnull"`
	Email     string     `bun:"email,notnull"`
	Phone     string     `bun:"phone,notnull"`
	Password  string     `bun:"password,notnull"`
	Role      string     `bun:"role,notnull"`
	Enabled   bool       `bun:"enabled,default:true"`
	CreatedBy uuid.UUID  `bun:"created_by,type:uuid,notnull"`
	CreatedAt time.Time  `bun:"created_at,default:current_timestamp"`
	UpdatedAt time.Time  `bun:"updated_at,default:current_timestamp"`
	DeletedAt *time.Time `bun:"deleted_at,nullzero"`
}
