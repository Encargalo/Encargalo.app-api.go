package models

// DataEventBalanceRefundPending se publica al crear una solicitud de devolución
// y cada 5 minutos mientras status sea pending o accepted (sin completar).
// El consumer carga el resto de datos desde BD con request_id.
type DataEventBalanceRefundPending struct {
	MessageType    string `json:"type"`
	RequestID      string `json:"request_id"`
	MinutesPending int    `json:"minutes_pending"`
}
