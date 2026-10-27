package models

// DataEventOrderStatusChanged se publica cuando una orden cambia de estado
// (hoy: rechazo por el local y entrega confirmada). El consumidor de ETL lo
// usa solo como señal para re-sincronizar esa orden puntual contra OLAP — el
// payload no es la fuente de verdad, por eso solo lleva el id.
type DataEventOrderStatusChanged struct {
	MessageType string `json:"type"`
	OrderID     string `json:"order_id"`
	Status      string `json:"status"`
}

func (d *DataEventOrderStatusChanged) BuildFromOrder(order Order) {
	d.MessageType = "order_status_changed"
	d.OrderID = order.ID.String()
	d.Status = order.Status
}
