package dtos

import (
	"context"

	"github.com/go-playground/mold/v4/modifiers"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

var (
	validate = validator.New()
	conform  = modifiers.New()
)

type CreateOrder struct {
	ID            uuid.UUID `json:"id" validate:"required,uuid4" example:"3fa85f64-5717-4562-b3fc-2c963f66afa6"`
	ShopID        uuid.UUID `json:"shop_id" validate:"required,uuid4" example:"3fa85f64-5717-4562-b3fc-2c963f66afa6"`
	CustomerID    uuid.UUID `json:"customer_id" swaggerignore:"true"`
	MethodPayment string    `json:"method_payment" validate:"required,oneof=Nequi Efectivo" example:"Nequi"`
	Address       Address   `json:"address" validate:"required"`

	CreateItemsOrder []CreateItemsOrder `json:"items" validate:"required,dive,required"`
}

func (o *CreateOrder) Validate() error {

	_ = conform.Struct(context.Background(), o)

	return validate.Struct(o)
}
