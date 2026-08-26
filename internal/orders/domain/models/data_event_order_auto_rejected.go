package models

// DataEventOrderAutoRejected se publica cuando una orden Pending se rechaza
// sola al cumplirse PendingAcceptTimeout (el local no aceptó a tiempo).
type DataEventOrderAutoRejected struct {
	MessageType    string `json:"type"`
	OrderID        string `json:"order_id"`
	ShopID         string `json:"shop_id"`
	CustomerID     string `json:"customer_id"`
	MinutesPending int    `json:"minutes_pending"`
}

func (d *DataEventOrderAutoRejected) BuildFromOrder(order Order, minutesPending int) {
	d.MessageType = "order_auto_rejected"
	d.OrderID = order.ID.String()
	d.ShopID = order.ShopID.String()
	d.CustomerID = order.CustomerID.String()
	d.MinutesPending = minutesPending
}
