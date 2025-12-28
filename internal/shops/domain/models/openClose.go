package models

import (
	"Encargalo.app-api.go/internal/shops/domain/dtos"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type OpenClose struct {
	bun.BaseModel `bun:"table:business.shops"`

	ID     uuid.UUID `bun:"id,pk,type:uuid,default:gen_random_uuid()"`
	Opened bool      `bun:"opened,notnull"`
}

func (o *OpenClose) BuildToMoidel(status dtos.OpenCloseShop) {
	o.ID = status.ShopID
	o.Opened = status.Open
}
