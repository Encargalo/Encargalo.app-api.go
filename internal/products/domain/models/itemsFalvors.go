package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type ItemsFlavor struct {
	bun.BaseModel `bun:"table:products.items_flavors,alias:if"`

	ID        uuid.UUID `bun:"id,pk,type:uuid,default:gen_random_uuid()" json:"id"`
	ItemID    uuid.UUID `bun:"item_id,notnull,type:uuid" json:"product_id"`
	FlavorID  uuid.UUID `bun:"flavor_id,notnull,type:uuid" json:"flavor_id"`
	CreatedAt time.Time `bun:"created_at,default:current_timestamp" json:"created_at"`

	Items  *Item   `bun:"rel:belongs-to,join:item_id=id" json:"product,omitempty"`
	Flavor *Flavor `bun:"rel:belongs-to,join:flavor_id=id" json:"flavor,omitempty"`
}
