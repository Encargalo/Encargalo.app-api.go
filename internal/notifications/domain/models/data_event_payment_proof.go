package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/uptrace/bun"
)

type DataEventPaymentProofPending struct {
	MessageType    string
	ReviewID       string
	OrderID        string
	MinutesPending int
}

func (d *DataEventPaymentProofPending) BuildFromXMessage(msg redis.XMessage) {
	d.MessageType = asString(msg.Values["type"])
	d.ReviewID = asString(msg.Values["review_id"])
	d.OrderID = asString(msg.Values["order_id"])
	d.MinutesPending = asInt(msg.Values["minutes_pending"])
}

// PaymentReview vive en payments.reviews desde que el API unificó pagos:
// reference_type distingue si el comprobante es de una orden o de un viaje.
type PaymentReview struct {
	bun.BaseModel `bun:"table:payments.reviews"`

	ID               uuid.UUID `bun:"id,pk"`
	ReferenceID      uuid.UUID `bun:"reference_id"`
	ReferenceType    string    `bun:"reference_type"`
	ProofURL         string    `bun:"proof_url"`
	ExpectedAmountBs float64   `bun:"expected_amount_bs"`
	BcvRate          float64   `bun:"bcv_rate"`
	Status           string    `bun:"status"`
	CreatedAt        time.Time `bun:"created_at"`
}

const (
	ReferenceTypeOrder = "order"
	ReferenceTypeTrip  = "trip"
)
