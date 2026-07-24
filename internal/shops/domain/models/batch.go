package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// BatchBlockingOrderStatuses son estados que impiden cerrar el lote de un local.
// Strings literales para no acoplar shops → orders.
var BatchBlockingOrderStatuses = []string{
	"Payment Review",
	"Pending",
	"Accepted",
	"In Preparation",
	"Ready",
	"On The Way",
}

type Batch struct {
	bun.BaseModel `bun:"table:business.batches"`

	ID        uuid.UUID  `bun:"id,pk,type:uuid,default:gen_random_uuid()"`
	ShopID    uuid.UUID  `bun:"shop_id,type:uuid"`
	Batch     int        `bun:"batch"`
	Opened    bool       `bun:"opened,default:true"`
	CreatedAt time.Time  `bun:"created_at,nullzero"`
	CloseAt   *time.Time `bun:"close_at,nullzero"`
}
