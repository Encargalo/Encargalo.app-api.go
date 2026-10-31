package models

import "github.com/google/uuid"

// DataEventShipment es el evento de envío que consume el servicio de notificaciones.
type DataEventShipment struct {
	MessageType   string  `json:"type"`
	ShipmentID    string  `json:"shipment_id"`
	CustomerID    string  `json:"customer_id"`
	FareUSD       float64 `json:"fare_usd"`
	MethodPayment string  `json:"method_payment"`
}

func (d *DataEventShipment) BuildFromShipment(messageType string, shipment Shipment) {
	d.MessageType = messageType
	d.ShipmentID = shipment.ID.String()
	d.CustomerID = shipment.CustomerID.String()
	if shipment.Fare != nil {
		d.FareUSD = *shipment.Fare
	}
	d.MethodPayment = shipment.MethodPayment
}

const (
	EventShipmentCreated       = "shipment_created"
	EventShipmentNoDriver      = "shipment_no_driver"
	EventShipmentStatusChanged = "shipment_status_changed"
)

// DataEventShipmentPaymentProofPending usa el mismo type que trips/orders: el
// consumidor distingue por reference_type del comprobante.
type DataEventShipmentPaymentProofPending struct {
	MessageType      string  `json:"type"`
	ReviewID         string  `json:"review_id"`
	ReferenceType    string  `json:"reference_type"`
	ShipmentID       string  `json:"shipment_id"`
	ExpectedAmountBs float64 `json:"expected_amount_bs"`
	ProofURL         string  `json:"proof_url"`
	MinutesPending   int     `json:"minutes_pending"`
}

func (d *DataEventShipmentPaymentProofPending) BuildFrom(shipmentID, reviewID uuid.UUID, expectedAmountBs float64, proofURL string, minutesPending int) {
	d.MessageType = "payment_proof_pending"
	d.ReviewID = reviewID.String()
	d.ReferenceType = ReferenceTypeShipment
	d.ShipmentID = shipmentID.String()
	d.ExpectedAmountBs = expectedAmountBs
	d.ProofURL = proofURL
	d.MinutesPending = minutesPending
}
