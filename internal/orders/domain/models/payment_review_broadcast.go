package models

import (
	"time"

	"github.com/google/uuid"
)

// PaymentReviewBroadcast is the WS payload sent to staff CRM clients.
type PaymentReviewBroadcast struct {
	Type              string              `json:"type"`
	ReviewID          uuid.UUID           `json:"review_id"`
	OrderID           uuid.UUID           `json:"order_id"`
	ShopID            uuid.UUID           `json:"shop_id"`
	CustomerID        uuid.UUID           `json:"customer_id"`
	CustomerName      string              `json:"customer_name,omitempty"`
	CustomerPhone     string              `json:"customer_phone,omitempty"`
	ProofURL          string              `json:"proof_url"`
	ExpectedAmountBs  float64             `json:"expected_amount_bs"`
	ObservedAmountBs  *float64            `json:"observed_amount_bs,omitempty"`
	DifferenceBs      *float64            `json:"difference_bs,omitempty"`
	Status            PaymentReviewStatus `json:"status"`
	RejectionReason   *string             `json:"rejection_reason,omitempty"`
	CreditedBs        *float64            `json:"credited_bs,omitempty"`
	BalanceAmountBs   float64             `json:"balance_amount_bs,omitempty"`
	PagoMovilAmountBs float64             `json:"pago_movil_amount_bs,omitempty"`
	TotalUSD          float64             `json:"total_usd"`
	OrderStatus       string              `json:"order_status"`
	CreatedAt         time.Time           `json:"created_at"`
}
