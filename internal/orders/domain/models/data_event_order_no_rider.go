package models

type DataEventOrderNoRider struct {
	MessageType string `json:"type"`
	OrderID     string `json:"order_id"`
	ShopID      string `json:"shop_id"`
	CustomerID  string `json:"customer_id"`
	Total       int    `json:"total"`
}

func (d *DataEventOrderNoRider) BuildFromOrder(order Order) {
	d.MessageType = "order_no_rider"
	d.OrderID = order.ID.String()
	d.ShopID = order.ShopID.String()
	d.CustomerID = order.CustomerID.String()
	d.Total = order.TotalPrice
}
