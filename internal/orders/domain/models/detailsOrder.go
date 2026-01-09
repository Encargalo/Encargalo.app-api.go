package models

import (
	"encoding/json"

	"Encargalo.app-api.go/internal/customers/domain/models"
	"github.com/google/uuid"
)

type Details struct {
	CustomerDetails customerDetails `json:"customer"`
	ShopDetails     shopDetails     `json:"shop"`
	ItemsDetails    []itemsDetails  `json:"items"`
}

func (d *Details) BuildDetails(order *Order, shop *ShopOrder, customer *models.Accounts) {

	d.CustomerDetails.ID = customer.ID
	d.CustomerDetails.Name = customer.Name
	d.CustomerDetails.Phone = customer.Phone

	d.ShopDetails.ID = shop.ID
	d.ShopDetails.Name = shop.Name
	d.ShopDetails.Phone = shop.Phone
	d.ShopDetails.Logo = shop.Logo
	d.ShopDetails.Address = shop.Address
	d.ShopDetails.Latitude = shop.Latitude
	d.ShopDetails.Longitude = shop.Longitude

	d.ItemsDetails = make([]itemsDetails, len(order.ItemsOrders))
	for i := range order.ItemsOrders {
		d.ItemsDetails[i].ID = order.ItemsOrders[i].ID
		d.ItemsDetails[i].Name = order.ItemsOrders[i].Name
		d.ItemsDetails[i].Amount = order.ItemsOrders[i].Amount
		d.ItemsDetails[i].UnitPrice = order.ItemsOrders[i].UnitPrice
		d.ItemsDetails[i].TotalPrice = order.ItemsOrders[i].TotalPrice
		d.ItemsDetails[i].Observation = order.ItemsOrders[i].Observation
		d.ItemsDetails[i].Flavors = make(map[string]int)
		err := json.Unmarshal(order.ItemsOrders[i].Flavors, &d.ItemsDetails[i].Flavors)
		if err != nil {
			return
		}
		d.ItemsDetails[i].Additions = make([]additionsDetails, len(order.ItemsOrders[i].Additions))
		for j := range order.ItemsOrders[i].Additions {
			d.ItemsDetails[i].Additions[j].ID = order.ItemsOrders[i].Additions[j].AdditionID
			d.ItemsDetails[i].Additions[j].Name = order.ItemsOrders[i].Additions[j].Name
			d.ItemsDetails[i].Additions[j].Amount = order.ItemsOrders[i].Additions[j].Amount
			d.ItemsDetails[i].Additions[j].UnitPrice = order.ItemsOrders[i].Additions[j].UnitPrice
			d.ItemsDetails[i].Additions[j].TotalPrice = order.ItemsOrders[i].Additions[j].TotalPrice
		}
	}

}

type customerDetails struct {
	ID    uuid.UUID `json:"id"`
	Name  string    `json:"name"`
	Phone string    `json:"phone"`
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

type itemsDetails struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Amount      int       `json:"amount"`
	UnitPrice   int       `json:"unit_price"`
	TotalPrice  int       `json:"total_price"`
	Observation string    `json:"observation"`
	Flavors     map[string]int
	Additions   []additionsDetails `json:"additions"`
}

type additionsDetails struct {
	ID         uuid.UUID `json:"id"`
	Name       string    `json:"name"`
	Amount     int       `json:"amount"`
	UnitPrice  int       `json:"unit_price"`
	TotalPrice int       `json:"total_price"`
}
