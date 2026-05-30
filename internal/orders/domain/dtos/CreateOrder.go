package dtos

import (
	"context"
	"errors"

	"github.com/go-playground/mold/v4/modifiers"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

var (
	validate = validator.New()
	conform  = modifiers.New()
)

type PagoMovilData struct {
	Nombre   string `json:"nombre" validate:"required"`
	Cedula   string `json:"cedula" validate:"required"`
	Telefono string `json:"telefono" validate:"required"`
	Banco    string `json:"banco" validate:"required"`
}

type CreateOrder struct {
	ID                  uuid.UUID      `json:"id" validate:"required,uuid4" example:"3fa85f64-5717-4562-b3fc-2c963f66afa6"`
	ShopID              uuid.UUID      `json:"shop_id" validate:"required,uuid4" example:"3fa85f64-5717-4562-b3fc-2c963f66afa6"`
	CustomerID          uuid.UUID      `json:"customer_id" swaggerignore:"true"`
	MethodPayment       string         `json:"method_payment" validate:"required,oneof=EfectivoBs EfectivoDolar PagoMovil" example:"EfectivoBs"`
	PagoMovilData       *PagoMovilData `json:"pago_movil_data,omitempty"`
	PaymentProofImage   string         `json:"payment_proof_image,omitempty" example:"data:image/jpeg;base64,/9j/4AAQ..."`
	DeliveryFee         int            `json:"delivery_fee" validate:"required" example:"5000"`
	Address             Address        `json:"address" validate:"required"`

	CreateItemsOrder []CreateItemsOrder `json:"items" validate:"required,dive,required"`
}

func (o *CreateOrder) Validate() error {
	_ = conform.Struct(context.Background(), o)

	if err := validate.Struct(o); err != nil {
		return err
	}

	if o.MethodPayment == "PagoMovil" {
		if o.PagoMovilData == nil {
			return errors.New("pago_movil_data es requerido para el método de pago PagoMovil")
		}
		if err := validate.Struct(o.PagoMovilData); err != nil {
			return err
		}
		if o.PaymentProofImage == "" {
			return errors.New("payment_proof_image es requerido para el método de pago PagoMovil")
		}
	}

	return nil
}

type CreateOrderResponse struct {
	Message string `json:"message"`
}
