package models

import "github.com/google/uuid"

// DataEventTrip es el evento de viaje que consume el servicio de notificaciones.
type DataEventTrip struct {
	MessageType   string  `json:"type"`
	TripID        string  `json:"trip_id"`
	CustomerID    string  `json:"customer_id"`
	FareUSD       float64 `json:"fare_usd"`
	MethodPayment string  `json:"method_payment"`
}

func (d *DataEventTrip) BuildFromTrip(messageType string, trip Trip) {
	d.MessageType = messageType
	d.TripID = trip.ID.String()
	d.CustomerID = trip.CustomerID.String()
	if trip.Fare != nil {
		d.FareUSD = *trip.Fare
	}
	d.MethodPayment = trip.MethodPayment
}

const (
	EventTripCreated       = "trip_created"
	EventTripNoDriver      = "trip_no_driver"
	EventTripStatusChanged = "trip_status_changed"
)

// DataEventTripPaymentProofPending usa el mismo type que el de órdenes: el
// consumidor distingue por reference_type del comprobante.
type DataEventTripPaymentProofPending struct {
	MessageType      string  `json:"type"`
	ReviewID         string  `json:"review_id"`
	ReferenceType    string  `json:"reference_type"`
	TripID           string  `json:"trip_id"`
	ExpectedAmountBs float64 `json:"expected_amount_bs"`
	ProofURL         string  `json:"proof_url"`
	MinutesPending   int     `json:"minutes_pending"`
}

func (d *DataEventTripPaymentProofPending) BuildFrom(tripID, reviewID uuid.UUID, expectedAmountBs float64, proofURL string, minutesPending int) {
	d.MessageType = "payment_proof_pending"
	d.ReviewID = reviewID.String()
	d.ReferenceType = ReferenceTypeTrip
	d.TripID = tripID.String()
	d.ExpectedAmountBs = expectedAmountBs
	d.ProofURL = proofURL
	d.MinutesPending = minutesPending
}
