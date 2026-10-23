package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type PaymentReviewStatus string

const (
	ReviewPendingReview      PaymentReviewStatus = "pending_review"
	ReviewApproved           PaymentReviewStatus = "approved"
	ReviewOverpaid           PaymentReviewStatus = "overpaid"
	ReviewRejectedUnderpaid  PaymentReviewStatus = "rejected_underpaid"
	ReviewRejectedInvalid    PaymentReviewStatus = "rejected_invalid"
	ReviewRefundedShopReject PaymentReviewStatus = "refunded_shop_reject"
)

const (
	ReferenceTypeOrder = "order"
	ReferenceTypeTrip  = "trip"
)

const PaymentProofAlertInterval = 3 * time.Minute

// RejectionReasonUnderpaid es el motivo centinela que RejectUnderpaid pasa al
// subject. Los demás rechazos mandan el texto libre que escribió el staff.
const RejectionReasonUnderpaid = "underpaid"

type PaymentReview struct {
	bun.BaseModel `bun:"table:payments.reviews"`

	ID               uuid.UUID           `bun:"id,pk,type:uuid,default:gen_random_uuid()" json:"id"`
	ReferenceID      uuid.UUID           `bun:"reference_id,type:uuid,notnull" json:"reference_id"`
	ReferenceType    string              `bun:"reference_type,notnull" json:"reference_type"`
	ProofURL         string              `bun:"proof_url,notnull" json:"proof_url"`
	ExpectedAmountBs float64             `bun:"expected_amount_bs,notnull" json:"expected_amount_bs"`
	ObservedAmountBs *float64            `bun:"observed_amount_bs" json:"observed_amount_bs,omitempty"`
	DifferenceBs     *float64            `bun:"difference_bs" json:"difference_bs,omitempty"`
	BcvRate          float64             `bun:"bcv_rate,notnull" json:"bcv_rate"`
	Status           PaymentReviewStatus `bun:"status,notnull" json:"status"`
	RejectionReason  *string             `bun:"rejection_reason" json:"rejection_reason,omitempty"`
	StaffID          *uuid.UUID          `bun:"staff_id,type:uuid" json:"staff_id,omitempty"`
	CreditedBs       *float64            `bun:"credited_bs" json:"credited_bs,omitempty"`
	ReferenceLast4   *string             `bun:"reference_last4" json:"reference_last4,omitempty"`
	CreatedAt        time.Time           `bun:"created_at,notnull,default:now()" json:"created_at"`
	ResolvedAt       *time.Time          `bun:"resolved_at" json:"resolved_at,omitempty"`
	LastAlertedAt    *time.Time          `bun:"last_alerted_at,nullzero" json:"last_alerted_at,omitempty"`
}

// StaffReviewResult es genérico: payments/app no conoce Order ni Trip.
type StaffReviewResult struct {
	ReviewID        uuid.UUID `json:"review_id"`
	ReferenceID     uuid.UUID `json:"reference_id"`
	ReferenceType   string    `json:"reference_type"`
	Action          string    `json:"action"`
	DuplicateReject bool      `json:"duplicate_reject,omitempty"`
	RejectionReason string    `json:"rejection_reason,omitempty"`
}
