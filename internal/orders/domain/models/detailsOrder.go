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

type PaymentDetails struct {
	PaymentProofURL string `json:"payment_proof_url"`
}

// BuildDetails congela una foto del pedido para mostrarla luego (historial,
// recibo, panel del local) sin depender de que el catálogo original no haya
// cambiado. rate es la tasa BCV vigente al momento de crear la orden: se usa
// para fijar de una vez los montos en bolívares de cada item/adición, igual
// que Order.ApplyBsPrice hace con el total y el delivery fee.
func (d *Details) BuildDetails(order *Order, shop *ShopOrder, customer *models.Account, addressInstructions string, rate float64) {

	d.CustomerDetails.ID = customer.ID
	d.CustomerDetails.Name = customer.Name
	d.CustomerDetails.Phone = customer.Phone
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
