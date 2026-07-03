package models

import (
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// Flavor is the read model orders uses to resolve flavor names chosen on an
// item; it mirrors Addition rather than importing the products domain.
type Flavor struct {
	bun.BaseModel `bun:"table:products.flavors"`

	ID   uuid.UUID `bun:"id,pk,type:uuid,default:gen_random_uuid()" json:"id"`
	Name string    `bun:"name,notnull" json:"name"`
}
