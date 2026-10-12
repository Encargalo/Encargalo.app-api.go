package dtos

import (
	"context"

	"github.com/google/uuid"
)

type CreateItemsOrder struct {
	ItemID      uuid.UUID      `json:"item_id" validate:"required,uuid"`
	Amount      int            `json:"amount" validate:"required"`
	Observation string         `json:"observation"`
	Flavors     map[string]int `json:"flavors"`

	Additions []CreateAdditionsOrders `json:"additions"`
}

func (io *CreateItemsOrder) Validate() error {
	_ = conform.Struct(context.Background(), io)
	return validate.Struct(io)
}
