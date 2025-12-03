package models

import (
	"Encargalo.app-api.go/internal/orders/domain/dtos"
	"Encargalo.app-api.go/internal/products/domain/models"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Order struct {
	bun.BaseModel `bun:"table:orders.orders"`

	ID            uuid.UUID `bun:"id,pk,type:uuid"`
	ShopID        uuid.UUID `bun:"shop_id"`
	CustomerID    uuid.UUID `bun:"customer_id"`
	BatchID       uuid.UUID `bun:"batch_id,nullzero,type:uuid"`
	Number        int       `bun:"number"`
	Address       string    `bun:"address"`
	Latitude      float64   `bun:"latitude"`
	Longitude     float64   `bun:"longitude"`
	MethodPayment string    `bun:"method_payment"`
	Status        string    `bun:"status"`
	DeliveryFee   int       `bun:"delivery_fee"`
	TotalPrice    int       `bun:"total_price"`

	ItemsOrders []ItemsOrder `bun:"rel:has-many,join:id=order_id"`
}

func (o *Order) BuildDtoToModel(dto dtos.CreateOrder, batchID uuid.UUID) {
	o.ID = dto.ID
	o.ShopID = dto.ShopID
	o.CustomerID = dto.CustomerID
	o.BatchID = batchID
	o.Address = dto.Address.Address
	o.Latitude = dto.Address.Latitude
	o.Longitude = dto.Address.Longitude
	o.MethodPayment = dto.MethodPayment
	o.Status = "Pending"
	o.DeliveryFee = dto.DeliveryFee
	o.ItemsOrders = make([]ItemsOrder, len(dto.CreateItemsOrder))

	for i := range dto.CreateItemsOrder {
		o.ItemsOrders[i].BuildDtoToModel(dto.CreateItemsOrder[i], dto.ID)

	}

}

func (o *Order) GetItemsID() []uuid.UUID {

	ids := make([]uuid.UUID, len(o.ItemsOrders))
	for i := range o.ItemsOrders {
		ids[i] = o.ItemsOrders[i].ItemID
	}

	return ids

}

func (o *Order) SetItemName(items models.Items) {

	nameMap := make(map[uuid.UUID]string, len(items))
	for _, item := range items {
		nameMap[item.ID] = item.Name
	}

	for i := range o.ItemsOrders {
		if name, ok := nameMap[o.ItemsOrders[i].ItemID]; ok {
			o.ItemsOrders[i].Name = name
		}
	}
}

func (o *Order) SetPrices(items models.Items) {

	priceMap := make(map[uuid.UUID]int, len(items))
	for _, item := range items {
		priceMap[item.ID] = item.Price
	}

	var total int
	for i := range o.ItemsOrders {
		if price, ok := priceMap[o.ItemsOrders[i].ItemID]; ok {
			o.ItemsOrders[i].UnitPrice = price
			o.ItemsOrders[i].TotalPrice = price * o.ItemsOrders[i].Amount
			total += o.ItemsOrders[i].TotalPrice
		}
	}

	o.TotalPrice = total + o.DeliveryFee
}

func (o *Order) SetAdditionalsName(additions []Addition) {

	nameMap := make(map[uuid.UUID]string, len(additions))
	for _, addition := range additions {
		nameMap[addition.ID] = addition.Name
	}

	for i := range o.ItemsOrders {
		for j := range o.ItemsOrders[i].Additions {
			if name, ok := nameMap[o.ItemsOrders[i].Additions[j].AdditionID]; ok {
				o.ItemsOrders[i].Additions[j].Name = name
			}
		}
	}

}

func (o *Order) SetAdditionalsPrices(addittions []Addition) {

	priceMap := make(map[uuid.UUID]int, len(addittions))
	for _, addition := range addittions {
		priceMap[addition.ID] = addition.Price
	}

	var totalAdd int
	for i := range o.ItemsOrders {

		var subTotalAdd int

		for j := range o.ItemsOrders[i].Additions {

			if price, ok := priceMap[o.ItemsOrders[i].Additions[j].AdditionID]; ok {

				o.ItemsOrders[i].Additions[j].UnitPrice = price

				o.ItemsOrders[i].Additions[j].TotalPrice = price * o.ItemsOrders[i].Amount

				subTotalAdd += o.ItemsOrders[i].Additions[j].TotalPrice

			}
		}

		o.ItemsOrders[i].TotalPrice += subTotalAdd
		totalAdd += subTotalAdd

	}
	o.TotalPrice += totalAdd
}
