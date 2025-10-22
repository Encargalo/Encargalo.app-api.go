package models

type DataEventOrderCreated struct {
	MessageType string `json:"type"`
	ShopID      string `json:"shop_id"`
	CustomerID  string `json:"customer_id"`
	Total       int    `json:"total"`
}

func (data *DataEventOrderCreated) BuidlToModelEvent(order Order) {
	data.MessageType = "order_created"
	data.ShopID = order.ShopID.String()
	data.CustomerID = order.CustomerID.String()
	data.Total = order.TotalPrice
}
