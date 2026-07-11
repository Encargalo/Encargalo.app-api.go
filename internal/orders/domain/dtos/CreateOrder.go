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

type CreateOrder struct {
	ID                uuid.UUID `json:"id" validate:"required,uuid4" example:"3fa85f64-5717-4562-b3fc-2c963f66afa6"`
	ShopID            uuid.UUID `json:"shop_id" validate:"required,uuid4" example:"3fa85f64-5717-4562-b3fc-2c963f66afa6"`
	CustomerID        uuid.UUID `json:"customer_id" swaggerignore:"true"`
	MethodPayment     string    `json:"method_payment" validate:"required,oneof=PagoMovil" example:"PagoMovil"`
	PaymentProofImage string    `json:"payment_proof_image" example:"data:image/jpeg;base64,/9j/4AAQ..."`
	Address           Address   `json:"address" validate:"required"`
	// ExpectedTotalPrice es el total que el front mostró al cliente justo
	// antes de confirmar (catálogo/carrito). El backend siempre recalcula el
	// cobro real desde la base de datos; este campo solo sirve para detectar
	// si el precio quedó desactualizado (ver ErrPriceChanged, 409) y nunca se
	// usa para fijar el monto cobrado.
	ExpectedTotalPrice float64 `json:"expected_total_price" validate:"required,gt=0" example:"18.5"`

	CreateItemsOrder []CreateItemsOrder `json:"items" validate:"required,dive,required"`
}

func (o *CreateOrder) Validate() error {
	_ = conform.Struct(context.Background(), o)

	if err := validate.Struct(o); err != nil {
		return err
	}

	if o.PaymentProofImage == "" {
		return errors.New("payment_proof_image es requerido")
	}

	return nil
}

type CreateOrderResponse struct {
	Message            string  `json:"message"`
	TotalPrice         float64 `json:"total_price"`
	IsLaunchPromo      bool    `json:"is_launch_promo"`
	PromoSubsidyAmount float64 `json:"promo_subsidy_amount"`
}
