package models

// DataEventFareQuoteCreated avisa al ETL que hay una cotización de viaje
// nueva para resincronizar CubeFareQuote sin esperar al batch; las
// cotizaciones son inmutables, así que no hace falta un evento de "changed".
type DataEventFareQuoteCreated struct {
	MessageType string `json:"type"`
	QuoteID     string `json:"quote_id"`
}

func (d *DataEventFareQuoteCreated) BuildFromQuoteID(quoteID string) {
	d.MessageType = "fare_quote_created"
	d.QuoteID = quoteID
}
