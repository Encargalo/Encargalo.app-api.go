package models

import (
	"time"

	"github.com/google/uuid"
)

// PaymentReviewBroadcast is the WS payload sent to staff CRM clients.
// ShopID, CustomerID, CustomerName, CustomerPhone, BalanceAmountBs,
// PagoMovilAmountBs, TotalUSD and OrderStatus are filled only when the
// reference type is enriched by its domain (orders today).
type PaymentReviewBroadcast struct {
	Type              string              `json:"type"`
	ReviewID          uuid.UUID           `json:"review_id"`
	ReferenceID       uuid.UUID           `json:"reference_id"`
	ReferenceType     string              `json:"reference_type"`
	ShopID            uuid.UUID           `json:"shop_id,omitempty"`
	CustomerID        uuid.UUID           `json:"customer_id,omitempty"`
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
	TotalUSD          float64             `json:"total_usd,omitempty"`
	OrderStatus       string              `json:"order_status,omitempty"`
	CreatedAt         time.Time           `json:"created_at"`
}
