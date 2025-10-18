package models

import (
	"time"

	"Encargalo.app-api.go/internal/products/domain/dtos"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Flavors []Flavor

type Flavor struct {
	bun.BaseModel `bun:"table:products.flavors,alias:f"`

	ID          uuid.UUID  `bun:"id,pk,type:uuid,default:gen_random_uuid()" json:"id"`
	ShopID      uuid.UUID  `bun:"shop_id,notnull,type:uuid" json:"shop_id"`
	CategoryID  uuid.UUID  `bun:"category_id,notnull,type:uuid" json:"category_id"`
	Name        string     `bun:"name,notnull,unique" json:"name"`
	Description *string    `bun:"description,nullzero" json:"description,omitempty"`
	CreatedAt   time.Time  `bun:"created_at,default:current_timestamp" json:"created_at"`
	UpdatedAt   time.Time  `bun:"updated_at,default:current_timestamp" json:"updated_at"`
	DeletedAt   *time.Time `bun:"deleted_at,nullzero" json:"deleted_at,omitempty"`
}

func (f *Flavor) ToDomainDTO() dtos.FlavorResponse {
	return dtos.FlavorResponse{
		ID:   f.ID,
		Name: f.Name,
	}
}

func (f *Flavors) ToDomainDTO() dtos.FlavorsResponse {

	var flavors dtos.FlavorsResponse

	for _, v := range *f {
		flavors.Add(v.ToDomainDTO())
	}

	return flavors

}
