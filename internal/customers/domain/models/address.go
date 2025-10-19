package models

import (
	"time"

	"Encargalo.app-api.go/internal/customers/domain/dto"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Addresses []Address

type Address struct {
	bun.BaseModel `bun:"table:customers.address"`

	ID         uuid.UUID  `bun:"id,pk,type:uuid,default:uuid_generate_v4()"`
	CustomerID uuid.UUID  `bun:"customer_id,type:uuid,notnull"`
	Alias      string     `bun:"alias,notnull"`
	Address    string     `bun:"address,notnull"`
	Reference  string     `bun:"reference,notnull"`
	Latitude   float64    `bun:"latitude,notnull"`
	Longitude  float64    `bun:"longitude,notnull"`
	CreatedAt  time.Time  `bun:"created_at,default:now()"`
	UpdatedAt  time.Time  `bun:"updated_at,default:now()"`
	DeletedAt  *time.Time `bun:"deleted_at,soft_delete"`
}

func (a *Address) BuildToModel(customer_id uuid.UUID, address dto.Address) {

	a.CustomerID = customer_id
	a.Alias = address.Alias
	a.Address = address.Address
	a.Reference = address.Reference
	a.Latitude = address.Cords.Latitude
	a.Longitude = address.Cords.Longitude

}

func (a *Address) ToDomainDTO() dto.Address {
	return dto.Address{
		ID:        a.ID,
		Alias:     a.Alias,
		Address:   a.Address,
		Reference: a.Reference,
		Cords: dto.Coords{
			Latitude:  a.Latitude,
			Longitude: a.Longitude,
		},
	}
}

func (a *Addresses) ToDomainDTO() dto.Addresses {

	var address dto.Addresses

	for _, v := range *a {
		address.Add(v.ToDomainDTO())
	}

	return address
}
