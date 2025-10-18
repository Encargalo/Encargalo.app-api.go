package models

import (
	"time"

	"Encargalo.app-api.go/internal/products/domain/dtos"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Additions []Addition

type Addition struct {
	bun.BaseModel `bun:"table:products.additions"`

	ID        uuid.UUID `bun:"id,pk,type:uuid"`
	ShopID    uuid.UUID `bun:"shop_id,notnull"`
	Name      string    `bun:"name,notnull"`
	Price     int       `bun:"price,notnull"`
	CreatedAt time.Time `bun:"created_at,default:now()"`
	UpdatedAt time.Time `bun:"updated_at,default:now()"`
	DeletedAt time.Time `bun:"deleted_at,nullzero"`
}

func (a *Addition) ToDomainDTO() dtos.AdditionResponse {
	return dtos.AdditionResponse{
		ID:    a.ID,
		Name:  a.Name,
		Price: a.Price,
	}
}

func (a *Additions) ToDomainDTO() dtos.AdditionsResponse {

	var addtions dtos.AdditionsResponse

	for _, v := range *a {
		addtions.Add(v.ToDomainDTO())
	}

	return addtions

}
