package models

type DataEventOrderCreated struct {
	MessageType string  `json:"type"`
	OrderID     string  `json:"order_id"`
	ShopID      string  `json:"shop_id"`
	CustomerID  string  `json:"customer_id"`
	Total       float64 `json:"total"`
}

func (data *DataEventOrderCreated) BuidlToModelEvent(order Order) {
	data.MessageType = "order_created"
	data.OrderID = order.ID.String()
	data.ShopID = order.ShopID.String()
	data.CustomerID = order.CustomerID.String()
	data.Total = order.TotalPrice
}
