package shops

import (
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Schedule struct {
	bun.BaseModel `bun:"table:business.schedules" swaggerignore:"true"`

	ID      uuid.UUID `bun:"id,pk,type:uuid,default:uuid_generate_v4()"`
	ShopID  uuid.UUID `bun:"shop_id,type:uuid,notnull"`
	DayID   uuid.UUID `bun:"day_id,type:uuid,notnull"`
	Open    string    `bun:"open"`
	Close   string    `bun:"close"`
	Enabled bool      `bun:"enabled,default:true"`
}
