package models

// DataEventDeliveryQuoteCreated avisa al ETL que hay una cotización de
// delivery nueva para resincronizar CubeDeliveryQuote sin esperar al batch;
// las cotizaciones son inmutables, así que no hace falta un evento de "changed".
type DataEventDeliveryQuoteCreated struct {
	MessageType string `json:"type"`
	QuoteID     string `json:"quote_id"`
}

func (d *DataEventDeliveryQuoteCreated) BuildFromQuoteID(quoteID string) {
	d.MessageType = "delivery_quote_created"
	d.QuoteID = quoteID
}
