package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type RefundRequestStatus string

const (
	RefundPending   RefundRequestStatus = "pending"
	RefundAccepted  RefundRequestStatus = "accepted"
	RefundCompleted RefundRequestStatus = "completed"
	RefundRejected  RefundRequestStatus = "rejected"

	RefundRequestWindow = 15 * time.Minute
	RefundMinAmountUSD  = 1.0
)

type BalanceRefundRequest struct {
	bun.BaseModel `bun:"table:finance.balance_refund_requests"`

	ID              uuid.UUID           `bun:"id,pk,type:uuid"`
	CustomerID      uuid.UUID           `bun:"customer_id,type:uuid"`
	TransactionID   uuid.UUID           `bun:"transaction_id,type:uuid"`
	OrderID         *uuid.UUID          `bun:"order_id,nullzero,type:uuid"`
	AmountUSD       float64             `bun:"amount_usd"`
	Status          RefundRequestStatus `bun:"status"`
	PMPhone         string              `bun:"pm_phone"`
	PMBankCode      string              `bun:"pm_bank_code"`
	PMNationalID    string              `bun:"pm_national_id"`
	ProofURL        *string             `bun:"proof_url,nullzero"`
	StaffID         *uuid.UUID          `bun:"staff_id,nullzero,type:uuid"`
	RejectionReason *string             `bun:"rejection_reason,nullzero"`
	CreatedAt       time.Time           `bun:"created_at"`
	ExpiresAt       time.Time           `bun:"expires_at"` // fin de ventana de elegibilidad del crédito (informativo; la solicitud no auto-expira)
	AcceptedAt      *time.Time          `bun:"accepted_at,nullzero"`
	CompletedAt     *time.Time          `bun:"completed_at,nullzero"`
	ResolvedAt      *time.Time          `bun:"resolved_at,nullzero"`
	LastAlertedAt   *time.Time          `bun:"last_alerted_at,nullzero"`
}

const RefundAlertInterval = 5 * time.Minute

func IsRefundableCreditMovement(movementType string) bool {
	switch MovementType(movementType) {
	case MovementTypePaymentOverage, MovementTypePaymentPartialCredit, MovementTypeOrderRejectedRefund:
		return true
	default:
		return false
	}
}
