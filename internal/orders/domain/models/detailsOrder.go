package models

import (
	"Encargalo.app-api.go/internal/pkg/money"
	"Encargalo.app-api.go/internal/users/domain/models"
	"github.com/google/uuid"
)

type Details struct {
	CustomerDetails customerDetails `json:"customer"`
	ShopDetails     shopDetails     `json:"shop"`
	AddressDetails  addressDetails  `json:"address"`
	ItemsDetails    []itemsDetails  `json:"items"`
	PaymentDetails  *PaymentDetails `json:"payment,omitempty"`
}

// PaymentDetails congela el estado de la revisi?n de pago manual para que el
// cliente sepa, sin depender del WS de staff, cu?nto se esperaba, cu?nto se
// detect? en el comprobante, la diferencia (si pag? de m?s o de menos), el
// motivo si fue rechazado (comprobante inv?lido o pago incompleto) y cu?nto
// se le acredit? a su balance en esos casos. `Status` refleja el estado de
// la revisi?n (PaymentReviewStatus) para que el front distinga el motivo
// exacto sin llamadas adicionales.
type PaymentDetails struct {
	PaymentProofURL   string              `json:"payment_proof_url,omitempty"`
	Status            PaymentReviewStatus `json:"status,omitempty"`
	ExpectedAmountBs  float64             `json:"expected_amount_bs,omitempty"`
	ObservedAmountBs  *float64            `json:"observed_amount_bs,omitempty"`
	DifferenceBs      *float64            `json:"difference_bs,omitempty"`
	RejectionReason   *string             `json:"rejection_reason,omitempty"`
	CreditedBs        *float64            `json:"credited_bs,omitempty"`
	BalanceAmountBs   float64             `json:"balance_amount_bs,omitempty"`
	PagoMovilAmountBs float64             `json:"pago_movil_amount_bs,omitempty"`
	// PagoMovilAmountUSD se lee de pedidos viejos (pre-migracion a Bs). No se
	// emite en escrituras nuevas; paymentSplitFromDetails lo convierte con la
	// tasa congelada si PagoMovilAmountBs viene vacio.
	PagoMovilAmountUSD float64 `json:"pago_movil_amount_usd,omitempty"`
	// BcvRate es la tasa del quote/review congelada al crear la orden. Sirve
	// para reconstruir total_bs y el fallback de pago_movil_amount_usd.
	BcvRate float64 `json:"bcv_rate,omitempty"`
	// Campos para solicitar devolucion en efectivo desde el order_update WS
	// (misma info que GET /users/balance sobre el movimiento cash-refundable).
	TransactionID       *uuid.UUID `json:"transaction_id,omitempty"`
	RefundablePMBs      *float64   `json:"refundable_pm_bs,omitempty"`
	CanRequestRefund    bool       `json:"can_request_refund,omitempty"`
	RefundEligibleUntil *string    `json:"refund_eligible_until,omitempty"`
}

// BuildDetails congela una foto del pedido para mostrarla luego (historial,
// recibo, panel del local) sin depender de que el cat?logo original no haya
// cambiado. rate es la tasa BCV vigente al momento de crear la orden: se usa
// para fijar de una vez los montos en bol?vares de cada item/adici?n, igual
// que Order.ApplyBsPrice hace con el total y el delivery fee.
func (d *Details) BuildDetails(order *Order, shop *ShopOrder, customer *models.Account, addressInstructions string, rate float64) {

	d.CustomerDetails.ID = customer.ID
	d.CustomerDetails.Name = customer.Name
	if customer.Phone != nil {
		d.CustomerDetails.Phone = *customer.Phone
	}
	if customer.SurName != nil {
		d.CustomerDetails.LastName = *customer.SurName
	}

	d.ShopDetails.ID = shop.ID
	d.ShopDetails.Name = shop.Name
	d.ShopDetails.Phone = shop.Phone
	d.ShopDetails.Logo = shop.Logo
	d.ShopDetails.Address = shop.Address
	d.ShopDetails.Latitude = shop.Latitude
	d.ShopDetails.Longitude = shop.Longitude

	d.AddressDetails.Instructions = addressInstructions

	d.ItemsDetails = make([]itemsDetails, len(order.ItemsOrders))
	for i := range order.ItemsOrders {
		d.ItemsDetails[i].ID = order.ItemsOrders[i].ID
		d.ItemsDetails[i].Name = order.ItemsOrders[i].Name
		d.ItemsDetails[i].Amount = order.ItemsOrders[i].Amount
		d.ItemsDetails[i].Image = order.ItemsOrders[i].Image
		d.ItemsDetails[i].UnitPrice = order.ItemsOrders[i].UnitPrice
		d.ItemsDetails[i].TotalPrice = order.ItemsOrders[i].TotalPrice
		d.ItemsDetails[i].UnitPriceBs = money.RoundBs(float64(order.ItemsOrders[i].UnitPrice) * rate)
		d.ItemsDetails[i].TotalPriceBs = money.RoundBs(float64(order.ItemsOrders[i].TotalPrice) * rate)
		d.ItemsDetails[i].Observation = order.ItemsOrders[i].Observation
		d.ItemsDetails[i].Flavors = order.ItemsOrders[i].FlavorsDetails
		d.ItemsDetails[i].Additions = make([]additionsDetails, len(order.ItemsOrders[i].Additions))
		for j := range order.ItemsOrders[i].Additions {
			d.ItemsDetails[i].Additions[j].ID = order.ItemsOrders[i].Additions[j].AdditionID
			d.ItemsDetails[i].Additions[j].Name = order.ItemsOrders[i].Additions[j].Name
			d.ItemsDetails[i].Additions[j].Amount = order.ItemsOrders[i].Additions[j].Amount
			d.ItemsDetails[i].Additions[j].UnitPrice = order.ItemsOrders[i].Additions[j].UnitPrice
			d.ItemsDetails[i].Additions[j].TotalPrice = order.ItemsOrders[i].Additions[j].TotalPrice
			d.ItemsDetails[i].Additions[j].UnitPriceBs = money.RoundBs(float64(order.ItemsOrders[i].Additions[j].UnitPrice) * rate)
			d.ItemsDetails[i].Additions[j].TotalPriceBs = money.RoundBs(float64(order.ItemsOrders[i].Additions[j].TotalPrice) * rate)
		}
	}

}

type customerDetails struct {
	ID       uuid.UUID `json:"id"`
	Name     string    `json:"name"`
	LastName string    `json:"last_name,omitempty"`
	Phone    string    `json:"phone"`
}

type shopDetails struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Phone     string    `json:"phone"`
	Logo      string    `json:"logo"`
	Address   string    `json:"address"`
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
}

type addressDetails struct {
	Instructions string `json:"instructions,omitempty"`
}

type itemsDetails struct {
	ID           uuid.UUID          `json:"id"`
	Name         string             `json:"name"`
	Image        string             `json:"image"`
	Amount       int                `json:"amount"`
	UnitPrice    int                `json:"unit_price"`
	TotalPrice   int                `json:"total_price"`
	UnitPriceBs  float64            `json:"unit_price_bs"`
	TotalPriceBs float64            `json:"total_price_bs"`
	Observation  string             `json:"observation"`
	Flavors      []ItemFlavor       `json:"flavors"`
	Additions    []additionsDetails `json:"additions"`
}

type additionsDetails struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Amount       int       `json:"amount"`
	UnitPrice    int       `json:"unit_price"`
	TotalPrice   int       `json:"total_price"`
	UnitPriceBs  float64   `json:"unit_price_bs"`
	TotalPriceBs float64   `json:"total_price_bs"`
}
