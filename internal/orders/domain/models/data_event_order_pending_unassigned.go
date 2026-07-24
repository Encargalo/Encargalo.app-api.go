package models

// DataEventOrderPendingUnassigned se publica en los minutos 4 y 8 mientras
// la orden sigue en Pending sin aceptación del local.
type DataEventOrderPendingUnassigned struct {
	MessageType    string `json:"type"`
	OrderID        string `json:"order_id"`
	ShopID         string `json:"shop_id"`
	CustomerID     string `json:"customer_id"`
	MinutesPending int    `json:"minutes_pending"`
}

func (d *DataEventOrderPendingUnassigned) BuildFromOrder(order Order, minutesPending int) {
	d.MessageType = "order_pending_unassigned"
	d.OrderID = order.ID.String()
	d.ShopID = order.ShopID.String()
	d.CustomerID = order.CustomerID.String()
	d.MinutesPending = minutesPending
}
