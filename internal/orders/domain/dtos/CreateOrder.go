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
	ID         uuid.UUID `json:"id" validate:"required,uuid4" example:"3fa85f64-5717-4562-b3fc-2c963f66afa6"`
	CustomerID uuid.UUID `json:"customer_id" swaggerignore:"true"`
	// QuoteID congela el total en el servidor (POST /orders/quote). Si viene,
	// shop/address/items/expected_total_price se toman del quote (pueden omitirse).
	QuoteID           uuid.UUID `json:"quote_id,omitempty" example:"3fa85f64-5717-4562-b3fc-2c963f66afa6"`
	ShopID            uuid.UUID `json:"shop_id" example:"3fa85f64-5717-4562-b3fc-2c963f66afa6"`
	MethodPayment     string    `json:"method_payment" validate:"required,oneof=PagoMovil" example:"PagoMovil"`
	PaymentProofImage string    `json:"payment_proof_image" example:"data:image/jpeg;base64,/9j/4AAQ..."`
	Address           Address   `json:"address"`
	// ExpectedTotalPrice (legacy sin quote): total que el front mostró.
	// Con quote_id se ignora. Sin quote_id sigue siendo obligatorio.
	ExpectedTotalPrice float64 `json:"expected_total_price" example:"18.5"`
	// BalanceAmountBs es opcional, en bolívares (balance del customer es Bs).
	// 0 = no usar balance. Si > 0 se valida (con la tasa BCV del quote) que
	// equivalga a >= $1 y <= min(balance disponible, total de la orden). Si
	// cubre el total no se requiere comprobante.
	BalanceAmountBs float64 `json:"balance_amount_bs" example:"148.5"`

	CreateItemsOrder []CreateItemsOrder `json:"items"`
}

func (o *CreateOrder) Validate() error {
	_ = conform.Struct(context.Background(), o)

	if o.ID == uuid.Nil {
		return errors.New("id es requerido")
	}
	if o.MethodPayment != "PagoMovil" {
		return errors.New("method_payment inválido")
	}

	if o.QuoteID != uuid.Nil {
		return o.validateBalanceOnly()
	}

	if err := validate.Struct(o); err != nil {
		return err
	}
	if o.ShopID == uuid.Nil {
		return errors.New("shop_id es requerido")
	}
	if err := o.Address.Validate(); err != nil {
		return err
	}
	if len(o.CreateItemsOrder) == 0 {
		return errors.New("items es requerido")
	}
	for i := range o.CreateItemsOrder {
		if err := o.CreateItemsOrder[i].Validate(); err != nil {
			return err
		}
	}
	if o.ExpectedTotalPrice <= 0 {
		return errors.New("expected_total_price es requerido sin quote_id")
	}

	return o.validateBalanceOnly()
}

// validateBalanceOnly solo valida el signo: el mínimo equivalente a $1 y el
// tope contra el balance/total dependen de la tasa BCV del quote, y se
// resuelven en orderApp.CreateOrder (domain/dtos no tiene acceso a bcv).
func (o *CreateOrder) validateBalanceOnly() error {
	if o.BalanceAmountBs < 0 {
		return errors.New("balance_amount_bs no puede ser negativo")
	}
	return nil
}

type CreateOrderResponse struct {
	Message            string  `json:"message"`
	TotalPrice         float64 `json:"total_price"`
	IsLaunchPromo      bool    `json:"is_launch_promo"`
	PromoSubsidyAmount float64 `json:"promo_subsidy_amount"`
}
